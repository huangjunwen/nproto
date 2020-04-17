package binlogmsg

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/helpers/mycanal"
	"github.com/huangjunwen/nproto/helpers/mycanal/fulldump"
	"github.com/huangjunwen/nproto/helpers/mycanal/incrdump"
	sqlh "github.com/huangjunwen/nproto/helpers/sql"
	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/npmsg/enc"
	"github.com/huangjunwen/nproto/nproto/zlog"
)

const (
	// DefaultLockName is the default lock name for msg pipe.
	DefaultLockName = "nproto.binlogmsg"

	// DefaultMaxInflight is default max number of processings message.
	DefaultMaxInflight = 512

	// DefaultRetryWait is the default interval reconnection.
	DefaultRetryWait = 5 * time.Second
)

// BinlogMsgPipe is used to pipe messages from message tables to downstream.
type BinlogMsgPipe struct {
	downstream  nproto.MsgPublisher
	masterCfg   *mycanal.FullDumpConfig
	slaveCfg    *mycanal.IncrDumpConfig
	tableFilter MsgTableFilter
	decoder     enc.MsgPayloadDecoder
	lockName    string
	logger      zerolog.Logger
	maxInflight int
	retryWait   time.Duration
}

// BinlogMsgPublisher 'publishes' msg to MySQL (>=8.0.2) binlog: it simply insert msg to a table.
type BinlogMsgPublisher struct {
	schema  string
	table   string
	q       sqlh.Queryer
	encoder enc.MsgPayloadEncoder
}

// MsgTableFilter returns true if a given table is a msg table.
type MsgTableFilter func(schema, table string) bool

// Option is option for BinlogMsgPipe.
type Option func(*BinlogMsgPipe) error

// PublisherOption is option for BinlogMsgPublisher.
type PublisherOption func(*BinlogMsgPublisher) error

var (
	_ nproto.MsgPublisher = (*BinlogMsgPublisher)(nil)
)

// NewBinlogMsgPipe creates a new BinlogMsgPipe.
func NewBinlogMsgPipe(
	downstream nproto.MsgPublisher,
	masterCfg *mycanal.FullDumpConfig,
	slaveCfg *mycanal.IncrDumpConfig,
	tableFilter MsgTableFilter,
	opts ...Option,
) (*BinlogMsgPipe, error) {

	ret := &BinlogMsgPipe{
		downstream:  downstream,
		masterCfg:   masterCfg,
		slaveCfg:    slaveCfg,
		tableFilter: tableFilter,
		decoder:     enc.PBMsgPayloadDecoder{},
		lockName:    DefaultLockName,
		logger:      zerolog.Nop(),
		maxInflight: DefaultMaxInflight,
		retryWait:   DefaultRetryWait,
	}
	OptLogger(&zlog.DefaultZLogger)(ret)

	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// Run the main loop (flush messages to downstream) until ctx done.
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

	// Create a connections for lock and delete msgs in msg tables.
	db, err := pipe.masterCfg.Client()
	if err != nil {
		pipe.logger.Error().Err(err).Msg("Open db failed")
		return err
	}
	defer db.Close()
	pipe.logger.Info().Msg("Open db ok")

	// Get lock.
	{
		conn, err := db.Conn(ctx)
		if err != nil {
			pipe.logger.Error().Err(err).Msg("Get conn failed")
			return err
		}
		defer conn.Close()

		ok, err := getLock(ctx, conn, pipe.lockName)
		if err != nil || !ok {
			pipe.logger.Error().Err(err).Msg("Get lock failed")
			return err
		}
		defer releaseLock(ctx, conn, pipe.lockName)
	}
	pipe.logger.Info().Msg("Get lock ok")

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// queue of msgEntry
	mq := newTaskQ()
	defer func() {
		mq.Push(msgEntry(nil))
	}()

	// speed control
	c := make(chan struct{}, pipe.maxInflight)

	// post-process go routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		defer pipe.logger.Info().Msg("Post process go routine ended")

		for {
			entry := mq.Pop().(msgEntry)
			if entry == nil {
				// msgEntry(nil) indicates end
				break
			}

			// NOTE: Cancel ctx if error, but don't break loop until msgEntry(nil).
			if err := entry.GetPublishErr(); err != nil {
				cancel()
				pipe.logger.Error().Err(err).Uint64("msgId", entry.Id()).Str("msgSubj", entry.Subject()).Msg("Publish msg failed")
			} else {
				// Do not cancel delete, so we use context.Background().
				err := delMsg(context.Background(), db, entry.SchemaName(), entry.TableName(), entry.Id())
				if err != nil {
					cancel()
					pipe.logger.Error().Err(err).Uint64("msgId", entry.Id()).Str("msgSubj", entry.Subject()).Msg("Delete msg failed")
				}
			}

			// The entry is processed.
			<-c
		}
	}()

	pubCbWg := &sync.WaitGroup{}
	flush := func(entry msgEntry) error {
		select {
		case <-ctx.Done():
			return errors.WithMessage(ctx.Err(), "Context done during flush")

		case c <- struct{}{}:
		}

		pubCbWg.Add(1)
		pipe.flushMsgEntry(ctx, entry, func(err error) {
			entry.SetPublishErr(err)
			mq.Push(entry)
			pubCbWg.Done()
		})
		return nil
	}

	pipe.logger.Info().Msg("Full dump starting")

	// Use full dump to publish existent msgs.
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
		pipe.logger.Error().Err(err).Msg("Full dump ended with error")
		return err
	} else {
		pipe.logger.Info().Msg("Full dump ended")
	}

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
		pipe.logger.Error().Err(err).Msg("Incr dump ended with error")
	} else {
		pipe.logger.Info().Err(err).Msg("Incr dump ended")
	}
	return err
}

func (pipe *BinlogMsgPipe) flushMsgEntry(ctx context.Context, entry msgEntry, cb func(error)) {
	// Recover MetaData and MsgData.
	payload := &enc.MsgPayload{}
	if err := pipe.decoder.DecodePayload(entry.Data(), payload); err != nil {
		// Should not happen.
		panic(err)
	}

	if payload.MD != nil {
		ctx = nproto.NewOutgoingContextWithMD(ctx, payload.MD)
	}

	// Use PublishAsync if downstream is MsgAsyncPublisher for higher throughput.
	switch downstream := pipe.downstream.(type) {
	case nproto.MsgAsyncPublisher:
		if err := downstream.PublishAsync(ctx, entry.Subject(), payload.MsgData, cb); err != nil {
			cb(err)
		}
	default:
		cb(downstream.Publish(ctx, entry.Subject(), payload.MsgData))
	}

}

// NewBinlogMsgPublisher creates a new BinlogMsgPublisher. schema/table is the message table
// to store messages.
func NewBinlogMsgPublisher(schema, table string, q sqlh.Queryer) (*BinlogMsgPublisher, error) {
	return &BinlogMsgPublisher{
		schema:  schema,
		table:   table,
		q:       q,
		encoder: enc.PBMsgPayloadEncoder{}, // TODO: option
	}, nil
}

// Publish implements nproto.MsgPublisher interface. MetaData attached `ctx` will be
// passed unmodified to downstream publisher.
func (p *BinlogMsgPublisher) Publish(ctx context.Context, subject string, msgData []byte) error {
	data, err := p.encoder.EncodePayload(&enc.MsgPayload{
		MsgData: msgData,
		MD:      nproto.MDFromOutgoingContext(ctx),
	})
	if err != nil {
		return err
	}

	return addMsg(ctx, p.q, p.schema, p.table, subject, data)
}
