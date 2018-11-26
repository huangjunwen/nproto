package dbstore

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

	// Test CreateSQL.
	{
		_, err := db.Exec(dialect.CreateSQL(table))
		assert.NoError(err)
	}

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
}
