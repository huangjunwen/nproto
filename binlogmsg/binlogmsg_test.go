package binlogmsg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/huangjunwen/golibs/mycanal"
	"github.com/rs/zerolog"
	// "github.com/huangjunwen/golibs/mycanal/fulldump"
	// "github.com/huangjunwen/golibs/mycanal/incrdump"
	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/v2/enc"
	"github.com/huangjunwen/nproto/v2/enc/rawenc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
)

func newLogger() logr.Logger {
	out := zerolog.NewConsoleWriter()
	out.TimeFormat = time.RFC3339
	out.Out = os.Stderr
	lg := zerolog.New(&out).With().Timestamp().Logger()
	return (*zerologr.Logger)(&lg)
}

func createMsgTable(db *sql.DB, schema, table string) {
	if err := CreateMsgTable(context.Background(), db, schema, table); err != nil {
		panic(err)
	}
}

func dropMsgTable(db *sql.DB, schema, table string) {
	_, err := db.Exec(fmt.Sprintf("DROP TABLE %s.%s", schema, table))
	if err != nil {
		panic(err)
	}
}

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

func newPublisher(db *sql.DB, schema, table string) MsgPublisherFunc {
	return NewPbJsonPublisher(db, schema, table)
}

func newRawPublisher(db *sql.DB, schema, table string) MsgPublisherFunc {
	return NewMsgPublisher(rawenc.DefaultRawEncoder, db, schema, table)
}

func decodeJsonRawData(rawData interface{}) interface{} {
	rd := rawData.(*rawenc.RawData)
	if rd.Format != enc.JsonFormat {
		panic(fmt.Errorf("Expect json format but got %s", rd.Format))
	}
	var v interface{}
	err := json.Unmarshal(rd.Bytes, &v)
	if err != nil {
		panic(err)
	}
	return v
}

func TestPipe(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestPipe.\n")
	var err error
	assert := assert.New(t)

	// Starts test mysql server.
	var res *tstmysql.Resource
	{
		res, err = tstmysql.Run(&tstmysql.Options{
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
		defer res.Close()
		log.Printf("MySQL server started.\n")
	}

	// Connect to test mysql server.
	var db *sql.DB
	{
		db, err = res.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	// Configs.
	cfg := mycanal.Config{
		Host:     "localhost",
		Port:     res.HostPort,
		User:     "root",
		Password: res.RootPassword,
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
	dbName := res.DBName

	// Params.
	tstMdKey := "mdKey"
	tstMdVal := "mdVal"
	tstCtx := npmd.NewOutgoingContextWithMD(context.Background(), npmd.NewMetaDataPairs(tstMdKey, tstMdVal))
	tstSpec := MustMsgSpec(
		"some.topic",
		func() interface{} { return new(string) },
	)
	tstMsg := new(string)
	*tstMsg = "112233"

	func() {
		log.Printf("\n")
		log.Printf(">>>> Test publishing.\n")
		tableName := "__test_publishing"
		createMsgTable(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)
		publisher := newPublisher(db, dbName, tableName)

		n := 10
		for i := 0; i < n; i++ {
			assert.NoError(publisher.Publish(tstCtx, tstSpec, tstMsg))
		}
		assert.Equal(n, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> Test flushing messages by fulldump.\n")
		tableName := "__test_flush_fulldump"
		createMsgTable(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)
		publisher := newPublisher(db, dbName, tableName)

		if err := publisher.Publish(tstCtx, tstSpec, tstMsg); err != nil {
			log.Panic(err)
		}
		assert.Equal(1, countMsgs(db, dbName, tableName))

		// Downstream consumes published messages and then stop runCtx.
		runCtx, runCancel := context.WithCancel(context.Background())
		downstream := MsgAsyncPublisherFunc(func(
			ctx context.Context,
			spec MsgSpec,
			msg interface{},
			cb func(error),
		) error {
			time.AfterFunc(time.Duration(rand.Int63n(20))*time.Millisecond, func() {
				// Check metadata.
				md := npmd.MDFromOutgoingContext(ctx)
				assert.NotNil(md)
				assert.Equal(tstMdVal, string(md.Values(tstMdKey)[0]))

				// Check msg.
				m := decodeJsonRawData(msg).(string)
				assert.Equal(*tstMsg, m)

				runCancel()
				cb(nil)
				log.Printf("Downstream callbacked %q %+q\n", spec.SubjectName(), m)
			})
			return nil
		})

		pipe, err := NewMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			PipeOptLogger(newLogger()),
		)
		if err != nil {
			log.Panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> Test flushing messages by incrdump.\n")
		tableName := "__test_flush_incrdump"
		createMsgTable(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)
		publisher := newPublisher(db, dbName, tableName)
		rawPublisher := newRawPublisher(db, dbName, tableName)

		if err := publisher.Publish(tstCtx, tstSpec, tstMsg); err != nil {
			log.Panic(err)
		}
		assert.Equal(1, countMsgs(db, dbName, tableName))

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
		downstream := MsgAsyncPublisherFunc(func(
			ctx context.Context,
			spec MsgSpec,
			msg interface{},
			cb func(error),
		) error {
			time.AfterFunc(time.Duration(rand.Int63n(20))*time.Millisecond, func() {
				// Check metadata.
				md := npmd.MDFromOutgoingContext(ctx)
				assert.NotNil(md)
				assert.Equal(tstMdVal, string(md.Values(tstMdKey)[0]))

				// Check msg.
				m := decodeJsonRawData(msg).(string)
				assert.Equal(*tstMsg, m)

				mu.Lock()
				n++
				cur := n
				mu.Unlock()
				cb(nil)
				log.Printf("Downstream callbacked %q %+q\n", spec.SubjectName(), msg)

				if cur <= 10 {
					if err := rawPublisher.Publish(ctx, spec, msg); err != nil {
						log.Panic(err)
					}
				} else {
					runCancel()
				}
			})
			return nil
		})

		pipe, err := NewMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			PipeOptLogger(newLogger()),
		)
		if err != nil {
			log.Panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> Test publish error.\n")
		tableName := "__test_publish_error"
		createMsgTable(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)
		publisher := newPublisher(db, dbName, tableName)

		if err := publisher.Publish(tstCtx, tstSpec, tstMsg); err != nil {
			log.Panic(err)
		}
		assert.Equal(1, countMsgs(db, dbName, tableName))

		runCtx, runCancel := context.WithCancel(context.Background())
		mu := &sync.Mutex{}
		n := 0
		downstream := MsgAsyncPublisherFunc(func(
			ctx context.Context,
			spec MsgSpec,
			msg interface{},
			cb func(error),
		) error {
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
					runCancel()
					log.Printf("Emulate publish success: %d\n", cur)
				}
			})
			return nil
		})

		pipe, err := NewMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			PipeOptLogger(newLogger()),
			PipeOptRetryWait(time.Second),
		)
		if err != nil {
			log.Panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(0, countMsgs(db, dbName, tableName))
	}()

	func() {
		log.Printf("\n")
		log.Printf(">>>> Test max inflight.\n")
		tableName := "__test_max_inflight"
		createMsgTable(db, dbName, tableName)
		defer dropMsgTable(db, dbName, tableName)
		publisher := newPublisher(db, dbName, tableName)

		n := 10
		for i := 0; i < n; i++ {
			assert.NoError(publisher.Publish(tstCtx, tstSpec, tstMsg))
		}
		assert.Equal(n, countMsgs(db, dbName, tableName))

		// Downstream callbacks block until runCtx done.
		runCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		downstream := MsgAsyncPublisherFunc(func(
			ctx context.Context,
			spec MsgSpec,
			msg interface{},
			cb func(error),
		) error {
			go func() {
				<-runCtx.Done()
				cb(nil)
				log.Printf("Emulate publish success after ctx done\n")
			}()
			return nil
		})

		pipe, err := NewMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool { return table == tableName },
			PipeOptLogger(newLogger()),
			// 1 message remains.
			PipeOptMaxInflight(n-1),
		)
		if err != nil {
			log.Panic(err)
		}

		pipe.Run(runCtx)
		assert.Equal(1, countMsgs(db, dbName, tableName))
	}()
}
