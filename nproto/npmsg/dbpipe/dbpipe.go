package dbpipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog"

	sqlh "github.com/huangjunwen/nproto/helpers/sql"
	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/npmsg/enc"
	"github.com/huangjunwen/nproto/nproto/taskrunner"
	"github.com/huangjunwen/nproto/nproto/zlog"
)

var (
	// DefaultMaxDelBulkSz is the default value of OptMaxDelBulkSz.
	DefaultMaxDelBulkSz = 1000
	// DefaultMaxInflight is the default value of OptMaxInflight.
	DefaultMaxInflight = 2048
	// DefaultMaxBuf is the default value of OptMaxBuf.
	DefaultMaxBuf = 512
	// DefaultRetryWait is the default value of OptRetryWait.
	DefaultRetryWait = 2 * time.Second
	// DefaultFlushWait is the default value of OptFlushWait.
	DefaultFlushWait = 15 * time.Second
)

var (
	// ErrMaxInflightAndBuf is returned if OptMaxBuf > OptMaxInflight.
	ErrMaxInflightAndBuf = errors.New("nproto.npmsg.dbpipe.DBMsgPublisherPipe: MaxBuf should be <= MaxInflight")
	// ErrUnknownDialect is returned if the dialect is not supported.
	ErrUnknownDialect = func(dialect string) error {
		return fmt.Errorf("nproto.npmsg.dbpipe.DBMsgPublisherPipe: Unknown dialect: %+q", dialect)
	}
)

// DBMsgPublisherPipe is used as a publisher pipeline from RDBMS to downstream publisher.
type DBMsgPublisherPipe struct {
	// Options.
	logger           zerolog.Logger
	maxDelBulkSz     int
	maxInflight      int
	maxBuf           int
	retryWait        time.Duration
	flushWait        time.Duration
	encoder          enc.MsgPayloadEncoder
	decoder          enc.MsgPayloadDecoder
	noRedeliveryLoop bool

	// Immutable fields.
	downstream nproto.MsgPublisher
	dialect    dbStoreDialect
	db         *sql.DB
	table      string
	runner     taskrunner.TaskRunner // To run TxPlugin.TxCommitted's flush.

	// Mutable fields.
	loopWg    sync.WaitGroup
	closeCtx  context.Context
	closeFunc context.CancelFunc
}

// DBMsgPublisher is used to "publish" messages to the database.
// Implements nproto.MsgPublisher interface.
type DBMsgPublisher struct {
	// Immutable fields.
	pipe  *DBMsgPublisherPipe
	q     sqlh.Queryer
	batch string // Batch id.

	// Mutable fields.
	mu      sync.Mutex
	n       int      // Number of published messages.
	bufMsgs *msgList // Buffered messages, no more than maxBuf.
}

type dbStoreDialect interface {
	CreateTable(ctx context.Context, q sqlh.Queryer, table string) error
	InsertMsg(ctx context.Context, q sqlh.Queryer, table, batch, subject string, data []byte) (id int64, err error)
	DeleteMsgs(ctx context.Context, q sqlh.Queryer, table string, ids []int64) error
	SelectMsgsByBatch(ctx context.Context, q sqlh.Queryer, table, batch string) msgStream
	SelectMsgsAll(ctx context.Context, q sqlh.Queryer, table string, tsDelta time.Duration) msgStream
	GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error)
	ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error
}

// Option is used when createing DBMsgPublisherPipe.
type Option func(*DBMsgPublisherPipe) error

// TxPlugin is returned from DBMsgPublisherPipe.TxPlugin.
type TxPlugin struct {
	pipe      *DBMsgPublisherPipe
	publisher *DBMsgPublisher
}

var (
	_ nproto.MsgPublisher = (*DBMsgPublisher)(nil)
	_ sqlh.TxPlugin       = (*TxPlugin)(nil)
)

// NewDBMsgPublisherPipe creates a new DBMsgPublisherPipe.
// `downstream` can be either nproto.MsgPublisher/nproto.MsgAsyncPublisher.
// `dialect` can be: "mysql".
// `db` is the database where to store messages.
// `table` is the name of the table to store messages.
// If `OptNoRedeliveryLoop` is given in `opts` then the redelivery loop will not be run.
func NewDBMsgPublisherPipe(downstream nproto.MsgPublisher, dialect string, db *sql.DB, table string, opts ...Option) (*DBMsgPublisherPipe, error) {
	ret := &DBMsgPublisherPipe{
		logger:       zerolog.Nop(),
		maxDelBulkSz: DefaultMaxDelBulkSz,
		maxInflight:  DefaultMaxInflight,
		maxBuf:       DefaultMaxBuf,
		retryWait:    DefaultRetryWait,
		flushWait:    DefaultFlushWait,
		encoder:      enc.PBMsgPayloadEncoder{}, // TODO: option
		decoder:      enc.PBMsgPayloadDecoder{}, // TODO: option
		downstream:   downstream,
		db:           db,
		table:        table,
		runner:       taskrunner.NewDefaultLimitedRunner(),
	}
	OptLogger(&zlog.DefaultZLogger)(ret)

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

	if err := ret.dialect.CreateTable(context.Background(), db, table); err != nil {
		return nil, err
	}

	ret.closeCtx, ret.closeFunc = context.WithCancel(context.Background())

	// Start the redelivery loop.
	if !ret.noRedeliveryLoop {
		ret.redeliveryLoop()
	}
	return ret, nil
}

// Close close the DBMsgPublisherPipe and wait the redeliveryLoop exits (if exists).
func (pipe *DBMsgPublisherPipe) Close() {
	pipe.closeFunc()
	pipe.loopWg.Wait()
	pipe.runner.Close()
}

// NewMsgPublisher creates a DBMsgPublisher. `q` must be connecting to the same database as DBMsgPublisherPipe.
func (pipe *DBMsgPublisherPipe) NewMsgPublisher(q sqlh.Queryer) *DBMsgPublisher {
	return &DBMsgPublisher{
		pipe:    pipe,
		q:       q,
		batch:   xid.New().String(),
		bufMsgs: &msgList{},
	}
}

// TxPlugin returns a sqlh.TxPlugin to use with sqlh.WithTx
func (pipe *DBMsgPublisherPipe) TxPlugin() *TxPlugin {
	return &TxPlugin{
		pipe: pipe,
	}
}

func (pipe *DBMsgPublisherPipe) redeliveryLoop() {
	fn := func() {
		// Wait for connection.
		conn, err := pipe.getConn()
		if err != nil {
			return
		}
		defer conn.Close()

		// Wait for global lock.
		err = pipe.getLock(conn)
		if err != nil {
			return
		}
		defer pipe.dialect.ReleaseLock(context.Background(), conn, pipe.table)

		// Main loop.
		for {
			// Verifies the connection to the database is still alive.
			err = conn.PingContext(context.Background())
			if err != nil {
				return
			}

			// Select msgs older than flushWait.
			stream := pipe.dialect.SelectMsgsAll(context.Background(), pipe.db, pipe.table, pipe.flushWait)
			pipe.flushMsgStream(pipe.closeCtx, stream)

			select {
			case <-pipe.closeCtx.Done():
				return
			case <-time.After(pipe.flushWait):
				continue
			}
		}
	}
	pipe.loopWg.Add(1)
	go func() {
		defer pipe.loopWg.Done()
		// Loop forever until closeCtx is done.
		for {
			select {
			case <-pipe.closeCtx.Done():
				return
			default:
			}
			fn()
		}
	}()
}

// getConn gets a single connection to db. It returns an error when closed.
func (pipe *DBMsgPublisherPipe) getConn() (conn *sql.Conn, err error) {
	for {
		conn, err = pipe.db.Conn(pipe.closeCtx)
		if err == nil {
			return
		}
		pipe.logger.Error().Err(err).Msg("Get db conn error")

		select {
		case <-pipe.closeCtx.Done():
			return nil, pipe.closeCtx.Err()
		case <-time.After(pipe.retryWait):
			continue
		}
	}
}

// getLock gets a global lock. It returns nil when lock acquired.
// And returns an error when error or closed.
func (pipe *DBMsgPublisherPipe) getLock(conn *sql.Conn) (err error) {
	for {
		acquired, err := pipe.dialect.GetLock(pipe.closeCtx, conn, pipe.table)
		if acquired {
			return nil
		}
		if err != nil {
			pipe.logger.Error().Err(err).Msg("Get db lock error")
			return err
		}

		select {
		case <-pipe.closeCtx.Done():
			return pipe.closeCtx.Err()
		case <-time.After(pipe.retryWait):
			continue
		}
	}
}

func (pipe *DBMsgPublisherPipe) flushMsgNode(ctx context.Context, node *msgNode, cb func(error)) {
	// Recover MetaData and MsgData.
	payload := &enc.MsgPayload{}
	if err := pipe.decoder.DecodePayload(node.Data, payload); err != nil {
		// Should not happen.
		panic(err)
	}

	if payload.MD != nil {
		ctx = nproto.NewOutgoingContextWithMD(ctx, payload.MD)
	}

	// Use PublishAsync if downstream is MsgAsyncPublisher for higher throughput.
	switch downstream := pipe.downstream.(type) {
	case nproto.MsgAsyncPublisher:
		if err := downstream.PublishAsync(ctx, node.Subject, payload.MsgData, cb); err != nil {
			cb(err)
		}
	default:
		cb(downstream.Publish(ctx, node.Subject, payload.MsgData))
	}
}

func (pipe *DBMsgPublisherPipe) flushMsgList(ctx context.Context, list *msgList) {
	if list.n == 0 {
		return
	}

	pubwg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	succList := msgList{}

	// Publish loop.
L:
	for {
		// Context maybe done during publishing.
		select {
		case <-ctx.Done():
			pipe.logger.Warn().Msg("Context done during publishing")
			break L
		case <-pipe.closeCtx.Done():
			pipe.logger.Warn().Msg("Close during publishing")
			break L
		default:
		}

		// Pop node.
		node := list.Pop()
		if node == nil {
			break
		}

		// Add publish task.
		pubwg.Add(1)
		cb := func(err error) {
			if err != nil {
				pipe.logger.Error().Err(err).Msg("Publish error")
			} else {
				// Append to success list.
				mu.Lock()
				succList.Append(node)
				mu.Unlock()
			}
			// Publish task done.
			pubwg.Done()
		}

		// Flush.
		pipe.flushMsgNode(ctx, node, cb)
	}

	// Wait all outgoing publishes done.
	pubwg.Wait()

	// Delete.
	pipe.deleteMsgList(&succList, nil)

	// Cleanup.
	list.Reset()
	succList.Reset()
}

func (pipe *DBMsgPublisherPipe) flushMsgStream(ctx context.Context, stream msgStream) {
	// This channel is used to control flush speed.
	taskc := make(chan struct{}, pipe.maxInflight)

	// This channel is used to sync with delete goroutine.
	delc := make(chan bool, 1)

	mu := &sync.Mutex{}
	succList := &msgList{}
	procList := &msgList{}

	// Start a separated goroutine for deleting msgs.
	delwg := &sync.WaitGroup{}
	delwg.Add(1)
	go func() {
		ok := true
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
			pipe.deleteMsgList(procList, taskc)

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
			pipe.logger.Warn().Msg("Context done during publishing")
			break L
		case <-pipe.closeCtx.Done():
			pipe.logger.Warn().Msg("Close during publishing")
			break L
		case taskc <- struct{}{}: // Speed control.
		}

		// Get msg.
		node, err := stream(true)
		if node == nil {
			if err != nil {
				pipe.logger.Error().Err(err).Msg("Msg stream error")
			}
			<-taskc
			break
		}

		// Add publish task.
		pubwg.Add(1)
		cb := func(err error) {
			if err != nil {
				<-taskc
				pipe.logger.Error().Err(err).Msg("Publish error")
			} else {
				// Append to success list.
				mu.Lock()
				succList.Append(node)
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

		// Flush.
		pipe.flushMsgNode(ctx, node, cb)
	}

	// Close the stream.
	stream(false)

	// Wait all outgoing publishes done.
	pubwg.Wait()

	// Close delc and wait delete goroutine end.
	close(delc)
	delwg.Wait()

	// Cleanup.
	succList.Reset()
	procList.Reset()
}

func (pipe *DBMsgPublisherPipe) deleteMsgList(list *msgList, taskc chan struct{}) {
	if list.n == 0 {
		return
	}
	iter := list.Iterate()
	end := false
	ids := []int64{}
	for !end {
		// Collect no more than maxDelBulkSz message ids.
		ids = ids[0:0]
		for len(ids) < pipe.maxDelBulkSz {
			node := iter()
			if node == nil {
				end = true
				break
			}
			ids = append(ids, node.Id)
		}

		// Delete.
		if err := pipe.dialect.DeleteMsgs(context.Background(), pipe.db, pipe.table, ids); err != nil {
			pipe.logger.Error().Err(err).Msg("Delete msg error")
		}

		// If there is a task channel, finish them.
		if taskc != nil {
			for _ = range ids {
				<-taskc
			}
		}
	}
}

// Publish implements nproto.MsgPublisher interface. MetaData attached `ctx` will be
// passed unmodified to downstream publisher.
func (p *DBMsgPublisher) Publish(ctx context.Context, subject string, msgData []byte) error {
	// Encode payload.
	data, err := p.pipe.encoder.EncodePayload(&enc.MsgPayload{
		MsgData: msgData,
		MD:      nproto.MDFromOutgoingContext(ctx),
	})
	if err != nil {
		return err
	}

	// First save to db.
	pipe := p.pipe
	id, err := pipe.dialect.InsertMsg(ctx, p.q, pipe.table, p.batch, subject, data)
	if err != nil {
		return err
	}

	// Append to bufMsgs if not more then maxBuf.
	p.mu.Lock()
	if p.n < pipe.maxBuf {
		node := &msgNode{
			Id:      id,
			Subject: subject,
			Data:    data,
		}
		p.bufMsgs.Append(node)
	}
	p.n += 1
	p.mu.Unlock()
	return nil
}

// Flush is used to flush messages to downstream. IMPORTANT: call this method
// ONLY after the transaction has been committed successfully.
func (p *DBMsgPublisher) Flush(ctx context.Context) {
	// Reset fields.
	p.mu.Lock()
	n := p.n
	bufMsgs := p.bufMsgs
	p.n = 0
	p.bufMsgs = &msgList{}
	p.mu.Unlock()
	defer bufMsgs.Reset()

	pipe := p.pipe
	// Small amount of messages. We can use the buffered messages directly.
	// Also no need to query db again.
	if bufMsgs.n == n {
		pipe.flushMsgList(ctx, bufMsgs)
		return
	}

	// For larger amount of messages, we need to query the db again.
	pipe.flushMsgStream(ctx, pipe.dialect.SelectMsgsByBatch(context.Background(), pipe.db, pipe.table, p.batch))

}

// TxInitialized implements sqlh.TxPlugin interface.
func (p *TxPlugin) TxInitialized(db *sql.DB, tx *sql.Tx) error {
	if p.pipe.db != db {
		panic(errors.New("TxInitialized: db must be the same as pipe.db"))
	}
	p.publisher = p.pipe.NewMsgPublisher(tx)
	return nil
}

// TxCommitted implements sqlh.TxPlugin interface.
func (p *TxPlugin) TxCommitted() {
	p.pipe.runner.Submit(func() {
		p.publisher.Flush(p.pipe.closeCtx)
	})
}

// TxFinalised implements sqlh.TxPlugin interface.
func (p *TxPlugin) TxFinalised() {
}

// Publisher returns the publisher for used inside the transaction.
func (p *TxPlugin) Publisher() nproto.MsgPublisher {
	return p.publisher
}
