package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/huangjunwen/tstsvc/mysql"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestMySQLDialect(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestMySQLDialect.\n")
	assert := assert.New(t)
	var err error

	// Starts the server.
	var res *tstmysql.Resource
	{
		res, err = tstmysql.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("MySQL server started.\n")
	}

	// Creates client.
	var db *sql.DB
	{
		db, err = res.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL connection created.\n")
	}

	// Creates the dialect.
	dialect := nxDialect{}

	// Creates two contexts.
	bgCtx := context.Background()
	doneCtx, doneFn := context.WithCancel(bgCtx)
	doneFn()

	table := "msg_store"

	// CreateMsgStoreTable.
	{
		err := dialect.CreateMsgStoreTable(doneCtx, db, table)
		assert.Error(err)

		err = dialect.CreateMsgStoreTable(bgCtx, db, table)
		assert.NoError(err)

		err = dialect.CreateMsgStoreTable(bgCtx, db, table)
		assert.NoError(err)
	}

	// InsertMsg.
	var (
		id      = xid.New().String()
		subject = "sub.sub"
		data    = []byte("data")
	)
	{
		tx, err := db.Begin()
		if err != nil {
			log.Panic(err)
		}
		defer tx.Rollback()

		err = dialect.InsertMsg(bgCtx, tx, table, id, subject, data)
		assert.NoError(err)

		err = dialect.InsertMsg(doneCtx, tx, table, xid.New().String(), subject, data)
		assert.Error(err)

		tx.Commit()

		var n sql.NullInt64
		assert.NoError(db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n))
		assert.True(n.Valid)
		assert.Equal(int64(1), n.Int64)
	}

	// GetLock/ReleaseLock.
	{
		conn1, err := db.Conn(bgCtx)
		if err != nil {
			log.Panic(err)
		}
		defer conn1.Close()

		conn2, err := db.Conn(bgCtx)
		if err != nil {
			log.Panic(err)
		}
		defer conn2.Close()

		// conn1 should acquire the lock.
		acquired, err := dialect.GetLock(bgCtx, conn1, table)
		assert.NoError(err)
		assert.True(acquired)

		// conn2 should not acquire the lock.
		acquired, err = dialect.GetLock(bgCtx, conn2, table)
		assert.NoError(err)
		assert.False(acquired)

		// conn1 release lock.
		assert.NoError(dialect.ReleaseLock(bgCtx, conn1, table))

		// now conn2 should acquire the lock.
		acquired, err = dialect.GetLock(bgCtx, conn2, table)
		assert.NoError(err)
		assert.True(acquired)

		// conn2 release lock.
		assert.NoError(dialect.ReleaseLock(bgCtx, conn2, table))
	}

	// SelectMsgs.
	{
		conn, err := db.Conn(bgCtx)
		if err != nil {
			log.Panic(err)
		}
		defer conn.Close()

		iter, err := dialect.SelectMsgs(bgCtx, conn, table, 0*time.Second)
		assert.NoError(err)

		i, s, d, err := iter(true)
		assert.NoError(err)
		assert.Equal(id, i)
		assert.Equal(subject, s)
		assert.Equal(data, d)

		i, s, d, err = iter(true)
		assert.NoError(err)
		assert.Equal("", i)
		assert.Equal("", s)
		assert.Equal([]byte(nil), d)

		i, s, d, err = iter(false)
		assert.NoError(err)
		assert.Equal("", i)
		assert.Equal("", s)
		assert.Equal([]byte(nil), d)
	}

	// DeleteMsgs.
	{
		var n sql.NullInt64
		assert.NoError(db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n))
		assert.True(n.Valid)
		assert.Equal(int64(1), n.Int64)

		assert.NoError(dialect.DeleteMsgs(bgCtx, db, table, []string{id}))

		assert.NoError(db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n))
		assert.True(n.Valid)
		assert.Equal(int64(0), n.Int64)
	}

}
