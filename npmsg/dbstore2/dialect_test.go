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

	{
		_, err := db.Exec(dialect.CreateSQL(table))
		assert.NoError(err)
	}

	var id1, id2 int64
	{
		id1, err = dialect.InsertMsg(ctx, db, table, batch, "subj1", []byte("data1"))
		assert.NoError(err)
		id2, err = dialect.InsertMsg(ctx, db, table, batch, "subj2", []byte("data2"))
		assert.NoError(err)
	}

	{
		var n int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
		assert.NoError(err)
		assert.Equal(2, n)
	}

	{
		err = dialect.DeleteMsgs(ctx, db, table, []int64{id1, id2})
		assert.NoError(err)
	}

	{
		var n int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
		assert.NoError(err)
		assert.Equal(0, n)
	}
}
