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

func clearMsgs(db *sql.DB, schema, table string) {
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s.%s", schema, table))
	if err != nil {
		panic(err)
	}
	n := countMsgs(db, schema, table)
	if n != 0 {
		panic(fmt.Errorf("There are %d msg(s) in %s.%s after delete", n, schema, table))
	}
}

func newPublisher(db *sql.DB, schema, table string) (*BinlogMsgPublisher, error) {
	if err := CreateMsgTable(context.Background(), db, schema, table); err != nil {
		return nil, err
	}
	publisher, err := NewBinlogMsgPublisher(schema, table, db)
	if err != nil {
		return nil, err
	}
	return publisher, nil
}

func TestBinlogMsgPipe(t *testing.T) {
	log.Printf("\n\n")
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
		publisher, err := newPublisher(db, dbName, tableName)
		if err != nil {
			log.Panic(err)
		}
		defer clearMsgs(db, dbName, tableName)

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
		publisher, err := newPublisher(db, dbName, tableName)
		if err != nil {
			log.Panic(err)
		}
		defer clearMsgs(db, dbName, tableName)

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
		publisher, err := newPublisher(db, dbName, tableName)
		if err != nil {
			log.Panic(err)
		}
		defer clearMsgs(db, dbName, tableName)

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
}
