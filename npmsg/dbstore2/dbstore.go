package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/huangjunwen/nproto/npmsg"
	//"github.com/rs/xid"
	"github.com/rs/zerolog"
)

var (
	errNotSet = errors.New("Not set")
)

type DBStore struct {
	logger         zerolog.Logger
	sampler        func() zerolog.Sampler
	deleteBulkSize int

	// Immutable fields.
	downstream npmsg.RawMsgAsyncPublisher
	db         *sql.DB
	table      string
	dialect    dbStoreDialect

	// Mutable fields.
	closeCtx  context.Context
	closeFunc context.CancelFunc
}

type DBPublisher struct {
	// Immutable fields.
	store *DBStore
	q     Queryer
	batch string // Batch id.

	// Mutable fields.
	mu      sync.Mutex
	n       int      // Number of msgs.
	bufMsgs *msgList // Buffered messages, no more than maxBuf.
}

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type dbStoreDialect interface {
	CreateSQL(table string) string
	InsertMsg(ctx context.Context, q Queryer, table, batch, subject string, data []byte) (id int64, err error)
	DeleteMsgs(ctx context.Context, q Queryer, table string, ids []int64) error
}

func (store *DBStore) flushMsgList(ctx context.Context, list *msgList) {
	if list.n == 0 {
		return
	}
	logger := store.logger.With().Str("fn", "flushMsgList").Logger()
	pubLogger := logger.Sample(store.sampler())
	delLogger := logger.Sample(store.sampler())

	// Publish loop.
	iter := list.Iterate()
	pubwg := &sync.WaitGroup{}
L:
	for {
		// Context maybe done during publishing.
		select {
		case <-ctx.Done():
			logger.Warn().Msg("context done during publishing")
			break L
		default:
		}

		msg := iter()
		if msg == nil {
			break
		}

		msg.Err = errNotSet
		pubwg.Add(1)
		cb := func(err error) {
			msg.Err = err
			if err != nil {
				pubLogger.Error().Err(err).Msg("publish error")
			}
			pubwg.Done()
		}

		if err := store.downstream.PublishAsync(ctx, msg.Subject, msg.Data, cb); err != nil {
			cb(err)
		}
	}

	// Wait all publish done.
	pubwg.Wait()

	// The second loop to delete published messages.
	// NOTE: This loop is always run even after ctx done, to avoid message redelivery.
	iter = list.Iterate()
	ids := []int64{}
	end := false
	for !end {
		// Collect no more than deleteBulkSize successful published message ids.
		ids = ids[0:0]
		for len(ids) < store.deleteBulkSize {
			msg := iter()
			if msg == nil {
				end = true
				break
			}
			if msg.Err == nil {
				ids = append(ids, msg.Id)
			}
		}

		// Delete.
		// NOTE: Use background context since ctx maybe already done here.
		if err := store.dialect.DeleteMsgs(context.Background(), store.db, store.table, ids); err != nil {
			delLogger.Error().Err(err).Msg("delete msg error")
		}
	}

}

func (store *DBStore) flushMsgStream(ctx context.Context, stream func() *msgNode) {
	logger := store.logger.With().Str("fn", "flushMsgStream").Logger()
	pubLogger := logger.Sample(store.sampler())
	delLogger := logger.Sample(store.sampler())

	c := make(chan bool, 1)
	mu := &sync.Mutex{}
	list := &msgList{}

	// Start a goroutine to delete messages.
	delwg := &sync.WaitGroup{}
	delwg.Add(1)
	go func() {
		ids := []int64{}
		ok := true
		for ok {
			// NOTE: When c is closed, ok will become false.
			// But we still need to run one more time.
			ok = <-c

			// Replace the list.
			mu.Lock()
			l := list
			list = &msgList{}
			mu.Unlock()

			if l.n == 0 {
				continue
			}

			// Loop to delete published messages.
			iter := l.Iterate()
			end := false
			for !end {
				// Collect no more than deleteBulkSize message ids.
				ids = ids[0:0]
				for len(ids) < store.deleteBulkSize {
					msg := iter()
					if msg == nil {
						end = true
						break
					}
					ids = append(ids, msg.Id)
				}

				// Delete.
				// NOTE: Use background context since ctx maybe already done here.
				if err := store.dialect.DeleteMsgs(context.Background(), store.db, store.table, ids); err != nil {
					delLogger.Error().Err(err).Msg("delete msg error")
				}
			}
		}
		delwg.Done()
	}()

	// Publish loop.
	pubwg := &sync.WaitGroup{}
L:
	for {
		// Context maybe done during publishing.
		select {
		case <-ctx.Done():
			logger.Warn().Msg("context done during publishing")
			break L
		default:
		}

		msg := stream()
		if msg == nil {
			break
		}

		pubwg.Add(1)
		cb := func(err error) {
			if err != nil {
				pubLogger.Error().Err(err).Msg("publish error")
				return
			}

			// Append to list when successful.
			mu.Lock()
			list.Append(msg)
			n := list.n
			mu.Unlock()

			// If list has enough item, kick the delete goroutine in non-blocking manner.
			if n >= store.deleteBulkSize {
				select {
				case c <- true:
				default:
				}
			}

			pubwg.Done()
		}

		if err := store.downstream.PublishAsync(ctx, msg.Subject, msg.Data, cb); err != nil {
			cb(err)
		}
	}

	// Wait all publish done.
	pubwg.Wait()

	// Close c to end delete goroutine.
	close(c)

	// Wait delete goroutine done.
	delwg.Wait()

}
