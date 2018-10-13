package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/huangjunwen/nproto/nproto/npmsg"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

var (
	DefaultRetryInterval = 5 * time.Second
	DefaultFlushInterval = 15 * time.Second
	DefaultBatch         = 500
)

var (
	ErrFlushed = errors.New("nproto.npmsg.dbstore.DBRawMsgPublisher: already flushed")
)

type DBMsgStore struct {
	downstream npmsg.RawMsgPublisher
	db         *sql.DB
	dialect    DBMsgStoreDialect
	table      string

	logger        zerolog.Logger
	retryInterval time.Duration
	flushInterval time.Duration
	batch         int

	stopC chan struct{}
}

type DBMsgStoreDialect interface {
	// CreateMsgStoreTable creates the message store table.
	CreateMsgStoreTable(ctx context.Context, q Queryer, table string) error

	// InsertMsg is used to save a message.
	InsertMsg(ctx context.Context, q Queryer, table string, id, subject string, data []byte) error

	// DeleteMsgs is used to delete published messages.
	DeleteMsgs(ctx context.Context, q Queryer, table string, ids []string) error

	// GetLock is used to acquire an application-level lock on the given table.
	GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error)

	// ReleaseLock is used to releases the lock acquired by GetLock.
	ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error

	// SelectMsgs is used to select all messages stored and older than window.
	// A message is returned for each call to iter(true), id == "" means there is no more messages or an error occurred.
	// Use iter(false) to close the iterator.
	SelectMsgs(ctx context.Context, conn *sql.Conn, table string, windown time.Duration) (
		iter func(next bool) (id, subject string, data []byte, err error),
		err error,
	)
}

type DBRawMsgPublisher struct {
	store    *DBMsgStore
	q        Queryer
	flushed  bool
	ids      []string
	subjects []string
	datas    [][]byte
}

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DBMsgStoreOption func(*DBMsgStore) error

var (
	_ npmsg.RawMsgPublisher = (*DBRawMsgPublisher)(nil)
)

func NewDBMsgStore(
	downstream npmsg.RawMsgPublisher,
	db *sql.DB,
	dialect DBMsgStoreDialect,
	table string,
	opts ...DBMsgStoreOption,
) (*DBMsgStore, error) {

	// Construct DBMsgStore and apply options.
	ret := &DBMsgStore{
		downstream:    downstream,
		db:            db,
		dialect:       dialect,
		table:         table,
		logger:        zerolog.Nop(),
		retryInterval: DefaultRetryInterval,
		flushInterval: DefaultFlushInterval,
		batch:         DefaultBatch,
		stopC:         make(chan struct{}),
	}
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}

	// Create table at the very beginning.
	if err := dialect.CreateMsgStoreTable(context.Background(), db, table); err != nil {
		return nil, err
	}

	// Starts the flush loop.
	ret.loop()

	return ret, nil
}

func (store *DBMsgStore) loop() {

	cfh.Go("DBMsgStore.loop", func() {

		logger := store.logger.With().Str("fn", "loop").Logger()
		bgctx := context.Background()
		table := store.table
		dialect := store.dialect

		for {
			// Check stopC.
			select {
			case <-store.stopC:
				break
			default:
			}

			// Wrap the following code into a function to use defers.
			// One sql.Conn's life starts here.
			func() {
				var (
					conn *sql.Conn
					err  error
				)

				// 1. Get a single connection to the db.
				for {
					conn, err = store.db.Conn(bgctx)
					if err == nil {
						break
					}
					logger.Error().Err(err).Msg("")

					// Wait and retry forever unless stopped.
					select {
					case <-store.stopC:
						return
					case <-time.After(store.retryInterval):
					}
				}
				defer conn.Close()

				// 2. Acquire application lock.
				for {
					acquired, err := dialect.GetLock(bgctx, conn, table)
					if acquired {
						break
					}
					if err != nil {
						logger.Error().Err(err).Msg("")
						// NOTE: If the connection is not alive, should start all over again.
						if conn.PingContext(bgctx) != nil {
							return
						}
					}

					// Wait and retry forever unless stopped.
					select {
					case <-store.stopC:
						return
					case <-time.After(store.retryInterval):
					}
				}
				defer dialect.ReleaseLock(bgctx, conn, table)

				// 3. Main loop.
				for {
					// 3.1. Flush all stored messages to downstream.
					if err := func() error {
						// 3.1.1 Get iterator.
						iter, err := dialect.SelectMsgs(bgctx, conn, table, store.flushInterval)
						if err != nil {
							return err
						}
						defer iter(false)

						// 3.1.2 Iterate messages in batch.
						for {
							ids := make([]string, 0)
							subjects := make([]string, 0)
							datas := make([][]byte, 0)

							// Get at most store.batch messages a time.
							for len(ids) < store.batch {
								id, subject, data, err := iter(true)
								if err != nil {
									return err
								}
								if id == "" {
									break
								}
								ids = append(ids, id)
								subjects = append(subjects, subject)
								datas = append(datas, data)
							}

							// No message.
							if len(ids) == 0 {
								return nil
							}

							// Flush to downstream.
							pubErrs, err := store.flushToDownstream(bgctx, ids, subjects, datas)
							if err != nil {
								return err
							}

							// Log.
							var (
								pubErr error // Only log one error.
								nErrs  int
							)
							for _, pubErr = range pubErrs {
								if pubErr != nil {
									nErrs++
								}
							}
							if nErrs > 0 {
								logger.Error().
									Int("nMsgs", len(ids)).
									Int("nErrs", nErrs).
									Err(pubErr).
									Msg("Publish errors")
							}
						}

						return nil
					}(); err != nil {
						logger.Error().Err(err).Msg("")
						// NOTE: If the connection is not alive, should start all over again.
						if conn.PingContext(bgctx) != nil {
							return
						}
					}

					// 3.2. Wait a while.
					select {
					case <-store.stopC:
						return
					case <-time.After(store.flushInterval):
					}
				}
				// Main loop ends here.

			}()
			// One sql.Conn's life ends here.
		}

	})
	// Go routine ends here.
}

func (store *DBMsgStore) flushToDownstream(ctx context.Context, ids, subjects []string, datas [][]byte) (
	pubErrs []error,
	err error,
) {
	// Nothing to do.
	if len(ids) == 0 {
		return
	}

	// Publish to downstream according by downstream's type.
	switch downstream := store.downstream.(type) {
	case npmsg.RawBatchMsgPublisher:
		pubErrs = downstream.PublishBatch(ctx, subjects, datas)
	default:
		// Fallback to use a loop.
		for i, subject := range subjects {
			pubErrs = append(pubErrs, downstream.Publish(ctx, subject, datas[i]))
		}
	}

	// Collect published messages' ids.
	doneIds := []string{}
	for i, pubErr := range pubErrs {
		if pubErr == nil {
			doneIds = append(doneIds, ids[i])
		}
	}

	// Delete published messages.
	if len(doneIds) != 0 {
		err = store.dialect.DeleteMsgs(ctx, store.db, store.table, doneIds)
	}
	return
}

func (store *DBMsgStore) Close() {
	close(store.stopC)
}

func (store *DBMsgStore) NewPublisher(q Queryer) *DBRawMsgPublisher {
	return &DBRawMsgPublisher{
		store: store,
		q:     q,
	}
}

func (p *DBRawMsgPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	// Can't publish after flushing.
	if p.flushed {
		return ErrFlushed
	}

	// First save to db for reliable.
	id := xid.New().String()
	if err := p.store.dialect.InsertMsg(ctx, p.q, p.store.table, id, subject, data); err != nil {
		return err
	}

	// Then save to internal buffers.
	p.ids = append(p.ids, id)
	p.subjects = append(p.subjects, subject)
	p.datas = append(p.datas, data)

	return nil
}

func (p *DBRawMsgPublisher) Flush(ctx context.Context) {
	// Can flush only once.
	if p.flushed {
		return
	}
	p.flushed = true

	// Flush to downstream.
	p.store.flushToDownstream(ctx, p.ids, p.subjects, p.datas)
	return
}
