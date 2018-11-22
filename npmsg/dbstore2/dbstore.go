package dbstore

import (
	"context"
	"database/sql"
	"sync"

	"github.com/huangjunwen/nproto/npmsg"
	//"github.com/rs/xid"
	//"github.com/rs/zerolog"
)

type DBStore struct {
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

func (store *DBStore) flushMsgList(list *msgList) {

	// TODO: logging

	if list.n == 0 {
		return
	}

	ctx := context.Background() // NOTE: This is a background job.
	wg := &sync.WaitGroup{}
	wg.Add(list.n)

	// The first loop to publish all messages asynchronously.
	iter := list.Iterate()
	for {
		msg := iter()
		if msg == nil {
			break
		}

		msg.Err = nil
		cb := func(err error) {
			msg.Err = err
			wg.Done()
		}

		if err := store.downstream.PublishAsync(ctx, msg.Subject, msg.Data, cb); err != nil {
			cb(err)
		}
	}

	// Wait.
	wg.Wait()

	// The second loop to delete published messages.
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
		store.dialect.DeleteMsgs(ctx, store.db, store.table, ids)
	}

}
