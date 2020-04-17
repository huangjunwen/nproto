package dbpipe

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"

	"github.com/huangjunwen/tstsvc/mysql"
	"github.com/stretchr/testify/assert"
)

func TestMySQLDialect(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestMySQLDialect.\n")
	var err error
	assert := assert.New(t)

	table := "msgstore"
	dialect := mysqlDialect{}
	batch := "xxxx"
	ctx := context.Background()

	var res *tstmysql.Resource
	{
		res, err = tstmysql.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("MySQL server started.\n")
	}

	var db *sql.DB
	{
		db, err = res.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	// Test CreateTable.
	assert.NoError(dialect.CreateTable(ctx, db, table))

	// Test InsertMsg.
	{
		_, err = dialect.InsertMsg(ctx, db, table, batch, "subj1", []byte("data1"))
		assert.NoError(err)
		_, err = dialect.InsertMsg(ctx, db, table, batch, "subj2", []byte("data2"))
		assert.NoError(err)
	}

	{
		var n int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
		assert.NoError(err)
		assert.Equal(2, n)
	}

	// Test SelectMsgsByBatch.
	ids := []int64{}
	{
		stream := dialect.SelectMsgsByBatch(ctx, db, table, batch)
		for {
			msg, err := stream(true)
			assert.NoError(err)
			if msg == nil {
				break
			}
			ids = append(ids, msg.Id)
		}
		msg, err := stream(false)
		assert.NoError(err)
		assert.Nil(msg)
		assert.Len(ids, 2)
	}

	// Test SelectMsgsAll
	{
		ids := []int64{}
		stream := dialect.SelectMsgsAll(ctx, db, table, 0)
		for {
			msg, err := stream(true)
			assert.NoError(err)
			if msg == nil {
				break
			}
			ids = append(ids, msg.Id)
		}
		msg, err := stream(false)
		assert.NoError(err)
		assert.Nil(msg)
		assert.Len(ids, 2)
	}

	// Test DeleteMsgs.
	{
		err = dialect.DeleteMsgs(ctx, db, table, ids)
		assert.NoError(err)
	}

	{
		var n int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
		assert.NoError(err)
		assert.Equal(0, n)
	}

	// Test GetLock/ReleaseLock.
	{
		conn1, err := db.Conn(ctx)
		if err != nil {
			log.Panic(err)
		}
		defer conn1.Close()

		conn2, err := db.Conn(ctx)
		if err != nil {
			log.Panic(err)
		}
		defer conn2.Close()

		// conn1 should acquire the lock.
		acquired, err := dialect.GetLock(ctx, conn1, table)
		assert.NoError(err)
		assert.True(acquired)

		// conn2 should not acquire the lock.
		acquired, err = dialect.GetLock(ctx, conn2, table)
		assert.NoError(err)
		assert.False(acquired)

		// conn1 release lock.
		assert.NoError(dialect.ReleaseLock(ctx, conn1, table))

		// now conn2 should acquire the lock.
		acquired, err = dialect.GetLock(ctx, conn2, table)
		assert.NoError(err)
		assert.True(acquired)

		// conn2 release lock.
		assert.NoError(dialect.ReleaseLock(ctx, conn2, table))
	}
}
