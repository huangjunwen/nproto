package dbstore

import (
	"context"
	"database/sql"
	"sync"

	"github.com/huangjunwen/nproto/npmsg"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

type DBStore struct {
	logger         zerolog.Logger
	deleteBulkSize int
	maxInflight    int
	maxBuf         int

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
	CreateSQL(table string) string
	InsertMsg(ctx context.Context, q Queryer, table, batch, subject string, data []byte) (id int64, err error)
	DeleteMsgs(ctx context.Context, q Queryer, table string, ids []int64) error
	SelectMsgsByBatch(ctx context.Context, q Queryer, table, batch string) msgStream
}

var (
	_ npmsg.RawMsgPublisher = (*DBPublisher)(nil)
)

func (store *DBStore) NewPublisher(q Queryer) *DBPublisher {

	return &DBPublisher{
		store:   store,
		q:       q,
		batch:   xid.New().String(),
		bufMsgs: &msgList{},
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
				deleteNode(msg)
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
				deleteNode(msg)
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
		// Collect no more than deleteBulkSize message ids.
		idsBuff = idsBuff[0:0]
		for len(idsBuff) < store.deleteBulkSize {
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
		msg := newNode()
		msg.Id = id
		msg.Subject = subject
		msg.Data = data
		p.bufMsgs.Append(msg)
	}
	p.n += 1
	p.mu.Unlock()
	return nil
}

func (p *DBPublisher) Finish(ctx context.Context, committed bool) {
	p.mu.Lock()
	n := p.n
	bufMsgs := p.bufMsgs
	p.n = 0
	p.bufMsgs = nil // Just set to nil since there should be no further Publish.
	p.mu.Unlock()
	defer bufMsgs.Reset()

	// Do nothing if rollbacked.
	if !committed {
		return
	}

	store := p.store
	// Small amount of messages. We can use the buffered messages directly.
	// Also no need to query db again.
	if n <= store.maxBuf {
		store.flushMsgList(ctx, bufMsgs)
		return
	}

	// For larger amount of messages, we need to query the db again.
	store.flushMsgStream(ctx, store.dialect.SelectMsgsByBatch(ctx, store.db, store.table, p.batch))

}
