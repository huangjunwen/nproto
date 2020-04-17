package binlogmsg

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/helpers/mycanal"
	"github.com/huangjunwen/nproto/nproto"
)

func countMsgs(db *sql.DB, schema, table string) int {
	r := &sql.NullInt64{}
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, table)).Scan(&r)
	if err != nil {
		panic(err)
	}
	if !r.Valid {
		panic(fmt.Errorf("Count of %s.%s is invalid", schema, table))
	}
	return int(r.Int64)
}

func dropMsgTable(db *sql.DB, schema, table string) {
	_, err := db.Exec(fmt.Sprintf("DROP TABLE %s.%s", schema, table))
	if err != nil {
		panic(err)
	}
}

func newPublisher(db *sql.DB, schema, table string) *BinlogMsgPublisher {
	if err := CreateMsgTable(context.Background(), db, schema, table); err != nil {
		panic(err)
	}
	publisher, err := NewBinlogMsgPublisher(schema, table, db)
	if err != nil {
		panic(err)
	}
	return publisher
}

func TestBinlogMsgPipe(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestBinlogMsgPipe.\n")
	var err error
	bgctx := context.Background()
	rand.Seed(time.Now().Unix())
	assert := assert.New(t)

	// Starts test mysql server.
	var resMySQL *tstmysql.Resource
	{
		resMySQL, err = tstmysql.Run(&tstmysql.Options{
			Tag: "8.0.2",
			BaseRunOptions: dockertest.RunOptions{
				Cmd: []string{
					"--gtid-mode=ON",
					"--enforce-gtid-consistency=ON",
					"--log-bin=/var/lib/mysql/binlog",
					"--server-id=1",
					"--binlog-format=ROW",
					"--binlog-row-image=full",
					"--binlog-row-metadata=full",
				},
			},
		})
		if err != nil {
			log.Panic(err)
		}
		defer resMySQL.Close()
		log.Printf("MySQL server started.\n")
	}

	// Connects to test mysql server.
	var db *sql.DB
	{
		db, err = resMySQL.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	// Configs.
	cfg := mycanal.Config{
		Host:     "localhost",
		Port:     resMySQL.HostPort,
		User:     "root",
		Password: resMySQL.RootPassword,
	}
	masterCfg := &mycanal.FullDumpConfig{
		Config: cfg,
	}
	slaveCfg := &mycanal.IncrDumpConfig{
		Config:   cfg,
		ServerId: 1001,
	}
	_ = masterCfg
	_ = slaveCfg
	dbName := resMySQL.DBName

	func() {
		log.Printf("\n")
		log.Printf(">>>> TestBinlogMsgPipe: test publishing.\n")
		tableName := "__msg1"
		publisher := newPublisher(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)

		n := 10
		for i := 0; i < n; i++ {
			assert.NoError(publisher.Publish(bgctx, "test", []byte("test")))
		}
		assert.Equal(n, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> TestBinlogMsgPipe: test flushing messages by fulldump.\n")
		tableName := "__msg2"
		publisher := newPublisher(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)

		mu := &sync.Mutex{}
		msgs := map[string][]byte{
			"abc": []byte("abcc"),
			"def": []byte("deff"),
			"123": []byte("1233"),
		}
		for subject, msgData := range msgs {
			assert.NoError(publisher.Publish(bgctx, subject, msgData))
		}
		assert.Equal(len(msgs), countMsgs(db, dbName, tableName))

		// Downstream consumes published messages and then stop runCtx
		runCtx, runCancel := context.WithCancel(context.Background())
		downstream := nproto.MsgAsyncPublisherFunc(func(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
			time.AfterFunc(time.Duration(rand.Int63n(20))*time.Millisecond, func() {
				mu.Lock()
				v, ok := msgs[subject]
				delete(msgs, subject)
				l := len(msgs)
				mu.Unlock()

				assert.True(ok)
				assert.Equal(v, msgData)
				if l == 0 {
					runCancel()
				}
				cb(nil)
				log.Printf("Downstream callbacked %q %+q\n", subject, msgData)
			})
			return nil
		})

		pipe, err := NewBinlogMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
		)
		if err != nil {
			panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> TestBinlogMsgPipe: test flushing messages by incrdump.\n")
		tableName := "__msg3"
		publisher := newPublisher(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)

		// The init message.
		assert.NoError(publisher.Publish(bgctx, "subj", []byte("msgData")))

		// NOTE: Publish another message after downstream getting a message. If the second (or later) message is delivered,
		// then it must be delivered by incrdump (binlog events), here is why:
		//
		// fulldump starts a transaction with consistent snapshot, so any changes after 'START TRANSACTION ...'
		// statement will not be observed inside that transaction. And the second (or later) message is definitely
		// published after 'START TRANSACTION ...' since it is triggered in the first message's callback. So it can be
		// delivered by binlog events only.
		runCtx, runCancel := context.WithCancel(context.Background())
		mu := &sync.Mutex{}
		n := 0
		downstream := nproto.MsgAsyncPublisherFunc(func(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
			time.AfterFunc(time.Duration(rand.Int63n(20))*time.Millisecond, func() {
				mu.Lock()
				n++
				cur := n
				mu.Unlock()
				cb(nil)
				log.Printf("Downstream callbacked %q %+q\n", subject, msgData)

				if cur <= 10 {
					assert.NoError(publisher.Publish(bgctx, subject, msgData))
				} else {
					runCancel()
				}
			})
			return nil
		})

		pipe, err := NewBinlogMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
		)
		if err != nil {
			panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> TestBinlogMsgPipe: test publish error.\n")
		tableName := "__msg4"
		publisher := newPublisher(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)

		// The init message.
		assert.NoError(publisher.Publish(bgctx, "subj", []byte("msgData")))
		assert.Equal(1, countMsgs(db, dbName, tableName))

		runCtx, runCancel := context.WithCancel(context.Background())
		mu := &sync.Mutex{}
		n := 0
		downstream := nproto.MsgAsyncPublisherFunc(func(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
			time.AfterFunc(time.Duration(rand.Int63n(20))*time.Millisecond, func() {
				mu.Lock()
				n++
				cur := n
				mu.Unlock()

				if cur <= 2 {
					cb(fmt.Errorf("Some error"))
					log.Printf("Emulate publish error: %d\n", cur)
				} else {
					cb(nil)
					log.Printf("Emulate publish success: %d\n", cur)
					runCancel()
				}
			})
			return nil
		})

		pipe, err := NewBinlogMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			OptRetryWait(500*time.Millisecond),
		)
		if err != nil {
			panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> TestBinlogMsgPipe: test max inflight.\n")
		tableName := "__msg5"
		publisher := newPublisher(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)

		n := 10
		for i := 0; i < n; i++ {
			assert.NoError(publisher.Publish(bgctx, "subj", []byte("msgData")))
		}
		assert.Equal(n, countMsgs(db, dbName, tableName))

		// Downstream callbacks block until runCtx done.
		runCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		downstream := nproto.MsgAsyncPublisherFunc(func(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
			go func() {
				<-runCtx.Done()
				cb(nil)
				log.Printf("Emulate publish success after ctx done\n")
			}()
			return nil
		})

		pipe, err := NewBinlogMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			// 1 message remains
			OptMaxInflight(n-1),
		)
		if err != nil {
			panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(1, countMsgs(db, dbName, tableName))
	}()
}
