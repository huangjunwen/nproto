package sqlh

import (
	"context"
	"database/sql"
	"log"
	"testing"

	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	"github.com/stretchr/testify/assert"
)

func TestWithTx(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestWithTx.\n")
	var err error
	assert := assert.New(t)

	// Starts test mysql server.
	var resMySQL *tstmysql.Resource
	{
		resMySQL, err = tstmysql.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer resMySQL.Close()
		log.Printf("MySQL server started.\n")
	}

	// Connects to test mysql server.
	var db *sql.DB
	{
		db, err = resMySQL.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	bgctx := context.Background()

	// Commit case.
	{
		events := []int{}
		err := WithTx(bgctx, db, func(ctx context.Context, tx *sql.Tx) error {
			MustCurTxContext(ctx).OnFinalised(func() {
				events = append(events, -1)
			})
			MustCurTxContext(ctx).OnFinalised(func() {
				events = append(events, -2)
			})
			MustCurTxContext(ctx).OnCommitted(func() {
				events = append(events, 1)
			})
			MustCurTxContext(ctx).OnCommitted(func() {
				events = append(events, 2)
			})
			return nil
		})
		assert.NoError(err)
		assert.Equal([]int{2, 1, -2, -1}, events)
	}

	// Rollback case.
	{
		events := []int{}
		err := WithTx(bgctx, db, func(ctx context.Context, tx *sql.Tx) error {
			MustCurTxContext(ctx).OnFinalised(func() {
				events = append(events, -1)
			})
			MustCurTxContext(ctx).OnFinalised(func() {
				events = append(events, -2)
			})
			MustCurTxContext(ctx).OnCommitted(func() {
				events = append(events, 1)
			})
			MustCurTxContext(ctx).OnCommitted(func() {
				events = append(events, 2)
			})
			return Rollback
		})
		assert.NoError(err)
		assert.Equal([]int{-2, -1}, events)
	}

	// Panic case.
	{
		events := []int{}
		assert.Panics(func() {
			WithTx(bgctx, db, func(ctx context.Context, tx *sql.Tx) error {
				MustCurTxContext(ctx).OnFinalised(func() {
					events = append(events, -1)
				})
				MustCurTxContext(ctx).OnFinalised(func() {
					events = append(events, -2)
				})
				MustCurTxContext(ctx).OnCommitted(func() {
					events = append(events, 1)
				})
				MustCurTxContext(ctx).OnCommitted(func() {
					events = append(events, 2)
				})
				panic("xxxx")
				return Rollback
			})
		})
		assert.Equal([]int{-2, -1}, events)

	}

	// Locals
	{
		val := "aaa"
		f := func(ctx context.Context) {
			assert.Equal(val, MustCurTxContext(ctx).Local(struct{}{}))
		}
		err := WithTx(bgctx, db, func(ctx context.Context, tx *sql.Tx) error {
			MustCurTxContext(ctx).SetLocal(struct{}{}, val)
			f(ctx)
			return nil
		})
		assert.NoError(err)

	}
}
