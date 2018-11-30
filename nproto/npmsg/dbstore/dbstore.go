package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/nproto/nproto/npmsg"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

var (
	DefaultMaxDelBulkSz = 1000
	DefaultMaxInflight  = 2048
	DefaultMaxBuf       = 512
	DefaultRetryWait    = 2 * time.Second
	DefaultFlushWait    = 15 * time.Second
)

var (
	ErrMaxInflightAndBuf = errors.New("nproto.npmsg.dbstore.DBStore: MaxBuf should be <= MaxInflight")
	ErrUnknownDialect    = func(dialect string) error {
		return fmt.Errorf("nproto.npmsg.dbstore.DBStore: Unknown dialect: %+q", dialect)
	}
)

type DBStore struct {
	logger           zerolog.Logger
	maxDelBulkSz     int
	maxInflight      int
	maxBuf           int
	createTable      bool
	retryWait        time.Duration
	flushWait        time.Duration
	noRedeliveryLoop bool

	// Immutable fields.
	downstream npmsg.RawMsgAsyncPublisher
	dialect    dbStoreDialect
	db         *sql.DB
	table      string

	// Mutable fields.
	loopWg    sync.WaitGroup
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
	n       int      // Number of published messages.
	bufMsgs *msgList // Buffered messages, no more than maxBuf.
}

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type dbStoreDialect interface {
	CreateTable(ctx context.Context, q Queryer, table string) error
	InsertMsg(ctx context.Context, q Queryer, table, batch, subject string, data []byte) (id int64, err error)
	DeleteMsgs(ctx context.Context, q Queryer, table string, ids []int64) error
	SelectMsgsByBatch(ctx context.Context, q Queryer, table, batch string) msgStream
	SelectMsgsAll(ctx context.Context, q Queryer, table string, tsDelta time.Duration) msgStream
	GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error)
	ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error
}

type Option func(*DBStore) error

var (
	_ npmsg.RawMsgPublisher = (*DBPublisher)(nil)
)

func NewDBStore(downstream npmsg.RawMsgAsyncPublisher, dialect string, db *sql.DB, table string, opts ...Option) (*DBStore, error) {
	ret := &DBStore{
		logger:       zerolog.Nop(),
		maxDelBulkSz: DefaultMaxDelBulkSz,
		maxInflight:  DefaultMaxInflight,
		maxBuf:       DefaultMaxBuf,
		retryWait:    DefaultRetryWait,
		flushWait:    DefaultFlushWait,
		downstream:   downstream,
		db:           db,
		table:        table,
	}

	switch dialect {
	case "mysql":
		ret.dialect = mysqlDialect{}
	default:
		return nil, ErrUnknownDialect(dialect)
	}

	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}

	if ret.maxBuf > ret.maxInflight {
		return nil, ErrMaxInflightAndBuf
	}

	if ret.createTable {
		if err := ret.dialect.CreateTable(context.Background(), db, table); err != nil {
			return nil, err
		}
	}

	ret.closeCtx, ret.closeFunc = context.WithCancel(context.Background())

	// Start the redelivery loop.
	if !ret.noRedeliveryLoop {
		ret.redeliveryLoop()
	}
	return ret, nil
}

func (store *DBStore) Close() {
	store.closeFunc()
	store.loopWg.Wait()
}

func (store *DBStore) NewPublisher(q Queryer) *DBPublisher {

	return &DBPublisher{
		store:   store,
		q:       q,
		batch:   xid.New().String(),
		bufMsgs: &msgList{},
	}
}

// redeliveryLoop is the redelivery loop.
func (store *DBStore) redeliveryLoop() {
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
			// Select msgs older than flushWait.
			stream := store.dialect.SelectMsgsAll(store.closeCtx, store.db, store.table, store.flushWait)
			store.flushMsgStream(store.closeCtx, stream)

			select {
			case <-store.closeCtx.Done():
				return
			case <-time.After(store.flushWait):
				continue
			}
		}
	}
	store.loopWg.Add(1)
	go func() {
		defer store.loopWg.Done()
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
func (store *DBStore) getConn() (conn *sql.Conn, err error) {
	logger := &store.logger
	for {
		conn, err = store.db.Conn(store.closeCtx)
		if err == nil {
			return
		}
		logger.Error().Str("fn", "getConn").Err(err).Msg("get db conn error")

		select {
		case <-store.closeCtx.Done():
			return nil, store.closeCtx.Err()
		case <-time.After(store.retryWait):
			continue
		}
	}
}

// getLock gets a global lock. It returns nil when lock acquired.
// And returns an error when error or closed.
func (store *DBStore) getLock(conn *sql.Conn) (err error) {
	logger := &store.logger
	for {
		acquired, err := store.dialect.GetLock(store.closeCtx, conn, store.table)
		if acquired {
			return nil
		}
		if err != nil {
			logger.Error().Str("fn", "getLock").Err(err).Msg("get db lock error")
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

func (store *DBStore) flushMsgList(ctx context.Context, list *msgList) {
	if list.n == 0 {
		return
	}
	logger := store.logger.With().Str("fn", "flushMsgList").Logger()

	// Publish loop.
	pubwg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	succList := msgList{}
L:
	for {
		// Context maybe done during publishing.
		select {
		case <-ctx.Done():
			logger.Warn().Msg("context done during publishing")
			break L
		case <-store.closeCtx.Done():
			logger.Warn().Msg("close during publishing")
			break L
		default:
		}

		// Pop msg.
		msg := list.Pop()
		if msg == nil {
			break
		}

		// Add publish task.
		pubwg.Add(1)
		cb := func(err error) {
			if err != nil {
				// Release the msg if err.
				logger.Error().Err(err).Msg("publish error")
			} else {
				// Append to success list.
				mu.Lock()
				succList.Append(msg)
				mu.Unlock()
			}
			// Publish task done.
			pubwg.Done()
		}

		// PublishAsync.
		if err := store.downstream.PublishAsync(ctx, msg.Subject, msg.Data, cb); err != nil {
			cb(err)
		}
	}

	// Wait all publish done.
	pubwg.Wait()

	// Delete.
	store.deleteMsgs(&succList, nil, nil)

	// Cleanup.
	list.Reset()
	succList.Reset()
}

func (store *DBStore) flushMsgStream(ctx context.Context, stream msgStream) {
	logger := store.logger.With().Str("fn", "flushMsgStream").Logger()

	// This channel is used to control flush speed.
	taskc := make(chan struct{}, store.maxInflight)

	// This channel is used to sync with delete goroutine.
	delc := make(chan bool, 1)

	mu := &sync.Mutex{}
	succList := &msgList{}
	procList := &msgList{}

	// Start a seperate goroutine for deleting msgs.
	delwg := &sync.WaitGroup{}
	delwg.Add(1)
	go func() {
		ok := true
		idsBuff := []int64{} // Reusable buffer.
		for ok {
			// NOTE: When delc is closed, ok will become false.
			// But we still need to run one more time.
			ok = <-delc

			// Swap succList and procList.
			mu.Lock()
			tmp := succList
			succList = procList
			procList = tmp
			mu.Unlock()

			// Delete msgs in procList.
			store.deleteMsgs(procList, idsBuff, taskc)

			// Cleanup.
			procList.Reset()
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
		case <-store.closeCtx.Done():
			logger.Warn().Msg("close during publishing")
			break L
		case taskc <- struct{}{}:
		}

		// Get msg.
		msg, err := stream(true)
		if msg == nil {
			if err != nil {
				logger.Error().Err(err).Msg("msg stream error")
			}
			<-taskc
			break
		}

		// Add publish task.
		pubwg.Add(1)
		cb := func(err error) {
			if err != nil {
				// Release the msg if err.
				<-taskc
				logger.Error().Err(err).Msg("publish error")
			} else {
				// Append to success list.
				mu.Lock()
				succList.Append(msg)
				mu.Unlock()
				// Kick delete goroutine in non-blocking manner.
				select {
				case delc <- true:
				default:
				}
			}
			// Publish task done.
			pubwg.Done()
		}

		// PublishAsync.
		if err := store.downstream.PublishAsync(ctx, msg.Subject, msg.Data, cb); err != nil {
			cb(err)
		}
	}

	// Close the stream.
	stream(false)

	// Wait all publish done.
	pubwg.Wait()

	// Close delc and wait delete goroutine end.
	close(delc)
	delwg.Wait()

	// Cleanup.
	succList.Reset()
	procList.Reset()
}

func (store *DBStore) deleteMsgs(list *msgList, idsBuff []int64, taskc chan struct{}) {
	if list.n == 0 {
		return
	}
	iter := list.Iterate()
	end := false
	for !end {
		// Collect no more than maxDelBulkSz message ids.
		idsBuff = idsBuff[0:0]
		for len(idsBuff) < store.maxDelBulkSz {
			msg := iter()
			if msg == nil {
				end = true
				break
			}
			idsBuff = append(idsBuff, msg.Id)
		}

		// Delete.
		if err := store.dialect.DeleteMsgs(context.Background(), store.db, store.table, idsBuff); err != nil {
			store.logger.Error().Err(err).Str("fn", "deleteMsgs").Msg("delete msg error")
		}

		// If there is a task channel, finish them.
		if taskc != nil {
			for _ = range idsBuff {
				<-taskc
			}
		}
	}
}

func (p *DBPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	// First save to db.
	store := p.store
	id, err := store.dialect.InsertMsg(ctx, p.q, store.table, p.batch, subject, data)
	if err != nil {
		return err
	}

	// Append to bufMsgs if not more then maxBuf.
	p.mu.Lock()
	if p.n < store.maxBuf {
		msg := &msgNode{
			Id:      id,
			Subject: subject,
			Data:    data,
		}
		p.bufMsgs.Append(msg)
	}
	p.n += 1
	p.mu.Unlock()
	return nil
}

func (p *DBPublisher) Flush(ctx context.Context) {
	p.mu.Lock()
	n := p.n
	bufMsgs := p.bufMsgs
	p.n = 0
	p.bufMsgs = &msgList{}
	p.mu.Unlock()
	defer bufMsgs.Reset()

	store := p.store
	// Small amount of messages. We can use the buffered messages directly.
	// Also no need to query db again.
	if bufMsgs.n == n {
		store.flushMsgList(ctx, bufMsgs)
		return
	}

	// For larger amount of messages, we need to query the db again.
	store.flushMsgStream(ctx, store.dialect.SelectMsgsByBatch(ctx, store.db, store.table, p.batch))

}
