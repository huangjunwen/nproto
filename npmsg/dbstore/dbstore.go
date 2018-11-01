package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/huangjunwen/nproto/npmsg"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

var (
	DefaultRetryWait = 2 * time.Second
	DefaultFlushWait = 15 * time.Second
	DefaultBatch     = 1000
)

var (
	ErrClosed = errors.New("nproto.npmsg.dbstore.DBMsgStore: Closed.")
)

type DBMsgStore struct {
	// Options.
	logger    zerolog.Logger
	retryWait time.Duration
	flushWait time.Duration
	batch     int

	// Immutable fields.
	downstream npmsg.RawMsgPublisher
	db         *sql.DB
	dialect    DBMsgStoreDialect
	table      string

	// Mutable fields.
	closeCtx  context.Context
	closeFunc context.CancelFunc
}

type DBRawMsgPublisher struct {
	// Immutable fields.
	store *DBMsgStore
	q     Queryer

	// Mutable fields.
	mu       sync.Mutex
	ids      []string
	subjects []string
	datas    [][]byte
}

type DBMsgStoreDialect interface {
	// InsertMsg is used to store a message.
	InsertMsg(ctx context.Context, q Queryer, table string, id, subject string, data []byte) error

	// DeleteMsgs is used to delete published messages.
	DeleteMsgs(ctx context.Context, q Queryer, table string, ids []string) error

	// CreateMsgStoreTable creates the message store table if not exists.
	CreateMsgStoreTable(ctx context.Context, q Queryer, table string) error

	// GetLock is used to acquire an global lock on the given table.
	GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error)

	// ReleaseLock is used to releases the lock acquired by GetLock.
	ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error

	// SelectMsgs is used to select all messages older than window.
	// A message is returned for each call to iter(true), id == "" means there is no more messages or an error occurred.
	// Use iter(false) to close the iterator.
	SelectMsgs(ctx context.Context, conn *sql.Conn, table string, window time.Duration) (
		iter func(next bool) (id, subject string, data []byte, err error),
		err error,
	)
}

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Option func(*DBMsgStore) error

var (
	_ npmsg.RawMsgPublisher = (*DBRawMsgPublisher)(nil)
)

func NewDBMsgStore(downstream npmsg.RawMsgPublisher, db *sql.DB, dialect DBMsgStoreDialect, table string, opts ...Option) (*DBMsgStore, error) {
	store := &DBMsgStore{
		logger:     zerolog.Nop(),
		retryWait:  DefaultRetryWait,
		flushWait:  DefaultFlushWait,
		batch:      DefaultBatch,
		downstream: downstream,
		db:         db,
		dialect:    dialect,
		table:      table,
	}
	store.closeCtx, store.closeFunc = context.WithCancel(context.Background())
	for _, opt := range opts {
		if err := opt(store); err != nil {
			return nil, err
		}
	}

	if err := dialect.CreateMsgStoreTable(context.Background(), db, table); err != nil {
		return nil, err
	}

	store.loop()
	return store, nil
}

func (store *DBMsgStore) NewPublisher(q Queryer) *DBRawMsgPublisher {
	return &DBRawMsgPublisher{
		store: store,
		q:     q,
	}
}

func (store *DBMsgStore) Close() {
	store.closeFunc()
}

func (store *DBMsgStore) loop() {
	fn := func() {
		// Wait for connection.
		conn, err := store.getConn()
		if err != nil {
			return
		}
		defer conn.Close()

		// Wait for global lock.
		err = store.getLock(conn)
		if err != nil {
			return
		}
		defer store.dialect.ReleaseLock(store.closeCtx, conn, store.table)

		// Main loop.
		for {
			err = store.flushAll(conn)
			if err != nil {
				return
			}

			select {
			case <-store.closeCtx.Done():
				return
			case <-time.After(store.flushWait):
				continue
			}
		}

	}
	go func() {
		// Loop forever until closeCtx is done.
		for {
			select {
			case <-store.closeCtx.Done():
				return
			default:
			}
			fn()
		}
	}()
}

// getConn gets a single connection to db. It returns an error when closed.
func (store *DBMsgStore) getConn() (conn *sql.Conn, err error) {
	logger := store.logger.With().Str("fn", "getConn").Logger()
	for {
		conn, err = store.db.Conn(store.closeCtx)
		if err == nil {
			return
		}
		logger.Error().Err(err).Msg("")

		select {
		case <-store.closeCtx.Done():
			return nil, store.closeCtx.Err()
		case <-time.After(store.retryWait):
			continue
		}
	}
}

// getLock gets a global lock. It returns nil when lock acquired.
// And returns an error when DBMsgStoreDialect.GetLock returns error or closed.
func (store *DBMsgStore) getLock(conn *sql.Conn) (err error) {
	logger := store.logger.With().Str("fn", "getLock").Logger()
	for {
		acquired, err := store.dialect.GetLock(store.closeCtx, conn, store.table)
		if acquired {
			return nil
		}
		if err != nil {
			logger.Error().Err(err).Msg("")
			return err
		}

		select {
		case <-store.closeCtx.Done():
			return store.closeCtx.Err()
		case <-time.After(store.retryWait):
			continue
		}
	}
}

// flushAll flushes all messages to downstream. It returns error when db op error or closed.
func (store *DBMsgStore) flushAll(conn *sql.Conn) (err error) {
	logger := store.logger.With().Str("fn", "flushAll").Logger()

	iter, err := store.dialect.SelectMsgs(store.closeCtx, conn, store.table, store.flushWait)
	if err != nil {
		logger.Error().Err(err).Msg("SelectMsgs error")
		return err
	}
	defer iter(false)

	ids := make([]string, 0)
	subjects := make([]string, 0)
	datas := make([][]byte, 0)
	for {
		// Reuse buffers.
		ids = ids[:0]
		subjects = subjects[:0]
		datas := datas[:0]

		// Collect no more than batch messages a time.
		for len(ids) < store.batch {
			id, subject, data, err := iter(true)
			if err != nil {
				logger.Error().Err(err).Msg("iter error")
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
		pubErrs, err := store.flush(store.closeCtx, ids, subjects, datas)

		// NOTE: Log only one publish error.
		for _, pubErr := range pubErrs {
			if pubErr != nil {
				logger.Error().Err(pubErr).Msg("publish error")
				break
			}
		}
		if err != nil {
			logger.Error().Err(err).Msg("flush error")
			return err
		}
	}

}

// flush publishes a batch of messages to downstream.
func (store *DBMsgStore) flush(ctx context.Context, ids, subjects []string, datas [][]byte) (errs []error, err error) {
	if len(ids) == 0 {
		return
	}

	// Publish to downstream according by downstream's type.
	switch downstream := store.downstream.(type) {
	case npmsg.RawBatchMsgPublisher:
		errs = downstream.PublishBatch(ctx, subjects, datas)
	default:
		// Fallback to use a loop.
		for i, subject := range subjects {
			errs = append(errs, downstream.Publish(ctx, subject, datas[i]))
		}
	}

	// Collect published messages' ids.
	doneIds := []string{}
	for i, err := range errs {
		if err == nil {
			doneIds = append(doneIds, ids[i])
		}
	}

	// Delete published messages.
	if len(doneIds) != 0 {
		err = store.dialect.DeleteMsgs(ctx, store.db, store.table, doneIds)
	}
	return
}

func (p *DBRawMsgPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	// Save to db for reliable.
	id := xid.New().String()
	if err := p.store.dialect.InsertMsg(ctx, p.q, p.store.table, id, subject, data); err != nil {
		return err
	}

	// Save to internal buffer.
	p.mu.Lock()
	p.ids = append(p.ids, id)
	p.subjects = append(p.subjects, subject)
	p.datas = append(p.datas, data)
	p.mu.Unlock()
	return nil
}

func (p *DBRawMsgPublisher) Flush(ctx context.Context) {
	// Get buffered messages.
	p.mu.Lock()
	ids := p.ids
	subjects := p.subjects
	datas := p.datas
	p.ids = nil
	p.subjects = nil
	p.datas = nil
	p.mu.Unlock()

	// Nothing to do.
	if len(ids) == 0 {
		return
	}

	// Flush to downstream.
	p.store.flush(ctx, ids, subjects, datas)
	return
}
