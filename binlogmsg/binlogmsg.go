package binlogmsg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/mycanal"
	"github.com/huangjunwen/golibs/mycanal/fulldump"
	"github.com/huangjunwen/golibs/mycanal/incrdump"
	"github.com/huangjunwen/golibs/sqlh"
	"google.golang.org/protobuf/proto"

	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nppbmsg "github.com/huangjunwen/nproto/v2/pb/msg"
)

type BinlogMsgPipe struct {
	// Immutable fields.
	masterCfg   *mycanal.FullDumpConfig
	slaveCfg    *mycanal.IncrDumpConfig
	tableFilter MsgTableFilter
	downstream  interface{} // msg.MsgPublisher or msg.MsgAsyncPublisher
	lockName    string
	logger      logr.Logger
	maxInflight int
	retryWait   time.Duration
}

type BinlogMsgPublisher struct {
	schema  string
	table   string
	q       sqlh.Queryer
	encoder npenc.Encoder
}

// MsgTableFilter returns true if a given table is a msg table.
type MsgTableFilter func(schema, table string) bool

func (pipe *BinlogMsgPipe) Run(ctx context.Context) (err error) {
	for {
		pipe.run(ctx)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pipe.retryWait):
		}
	}
}

func (pipe *BinlogMsgPipe) run(ctx context.Context) (err error) {

	logger := pipe.logger
	db, err := pipe.masterCfg.Client()
	if err != nil {
		logger.Error(err, "open db failed")
		return err
	}
	defer db.Close()
	logger.Info("open db ok")

	// Get lock.
	{
		conn, err := db.Conn(ctx)
		if err != nil {
			logger.Error(err, "get db conn failed")
			return err
		}
		defer conn.Close()

		ok, err := getLock(ctx, conn, pipe.lockName)
		if err != nil || !ok {
			logger.Error(err, "get lock failed")
			return err
		}
		defer releaseLock(ctx, conn, pipe.lockName)

		logger.Info("get lock ok")
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	entryCh := make(chan msgEntry, pipe.maxInflight) // for post process
	defer close(entryCh)

	ctrlCh := make(chan struct{}, pipe.maxInflight) // for speed control

	// Post process go routine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		logger.Info("post process go routine ended")

		conn, err := db.Conn(context.Background())
		if err != nil {
			logger.Error(err, "post process go routine get del msg conn error")
			return
		}
		defer conn.Close()

		// Loop until end.
		for entry := range entryCh {
			// NOTE: Cancel ctx if error, but don't break the loop until q is closed.
			err := entry.GetPublishErr()
			if err != nil {
				cancel()
				logger.Error(err, "publish failed", "msgId", entry.Id(), "msgSubj", entry.Subject())
			} else {
				err = delMsg(context.Background(), conn, entry.SchemaName(), entry.TableName(), entry.Id())
				if err != nil {
					cancel()
					logger.Error(err, "del msg error", "msgId", entry.Id(), "msgSubj", entry.Subject())
				}
			}
			// Put back quota.
			<-ctrlCh
		}
	}()

	pubCbWg := &sync.WaitGroup{}
	flush := func(entry msgEntry) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ctrlCh <- struct{}{}: // Try to get quota.
		}

		pubCbWg.Add(1)
		pipe.flushMsgEntry(ctx, entry, func(err error) {
			entry.SetPublishErr(err)
			select {
			case entryCh <- entry:
			default:
				// XXX: len(c) == len(q), so q <- entry should never block.
				panic(fmt.Errorf("Unexpected branch"))
			}
			pubCbWg.Done()
		})
		return nil
	}

	logger.Info("full dump starting")

	gtidSet, err := fulldump.FullDump(ctx, pipe.masterCfg, func(ctx context.Context, q sqlh.Queryer) error {
		schemas, tables, err := listMsgTables(ctx, q, pipe.tableFilter)
		if err != nil {
			return err
		}

		for i := 0; i < len(schemas); i++ {
			if err := func() error {
				schema := schemas[i]
				table := tables[i]
				iter, err := fulldump.FullTableQuery(ctx, q, schema, table)
				if err != nil {
					return err
				}
				defer iter(false)

				for {
					row, err := iter(true)
					if err != nil {
						return err
					}
					if row == nil {
						return nil
					}

					entry := newMsgEntry(schema, table, row)
					if err := flush(entry); err != nil {
						return err
					}
				}
			}(); err != nil {
				return err
			}
		}
		return nil
	})

	// Wait all outgoing publish callbacks done.
	// Note that Post process maybe not finished yet.
	pubCbWg.Wait()

	if err != nil {
		logger.Error(err, "full dump ended with error")
		return err
	}
	logger.Info("full dump ended")

	// Now start incr dump to capture changes.
	err = incrdump.IncrDump(ctx, pipe.slaveCfg, gtidSet, func(ctx context.Context, e interface{}) error {

		switch ev := e.(type) {
		case *incrdump.RowInsertion:
			schema := ev.SchemaName()
			table := ev.TableName()
			if !pipe.tableFilter(schema, table) {
				return nil
			}

			entry := newMsgEntry(schema, table, ev.AfterDataMap())
			if err := flush(entry); err != nil {
				return err
			}
		}

		return nil
	})

	// Wait all outgoing publish callbacks done.
	// Note that Post process maybe not finished yet.
	pubCbWg.Wait()

	if err != nil {
		logger.Error(err, "incr dump ended with error")
	} else {
		logger.Info("Incr dump ended")
	}

	return err
}

func (pipe *BinlogMsgPipe) flushMsgEntry(ctx context.Context, entry msgEntry, cb func(error)) {
	spec := MustRawDataMsgSpec(entry.Subject())

	msg := &nppbmsg.MessageWithMD{}
	if err := proto.Unmarshal(entry.Data(), msg); err != nil {
		// Should not happen.
		panic(err)
	}

	if len(msg.MetaData) != 0 {
		ctx = npmd.NewOutgoingContextWithMD(ctx, nppbmd.MetaData(msg.MetaData))
	}

	data := &npenc.RawData{
		Format: msg.MsgFormat,
		Bytes:  msg.MsgBytes,
	}

	// Use PublishAsync if downstream is MsgAsyncPublisher for higher throughput.
	switch downstream := pipe.downstream.(type) {
	case MsgAsyncPublisher:
		if err := downstream.PublishAsync(ctx, spec, data, cb); err != nil {
			cb(err)
		}

	case MsgPublisher:
		cb(downstream.Publish(ctx, spec, data))

	default:
		panic(fmt.Errorf("downstream %T is neither MsgPublisher nor MsgAsyncPublisher", pipe.downstream))
	}

}
