package dbstore

import (
	"context"
	"database/sql"
	"log"
	"strconv"
	"sync"
	"testing"

	"github.com/huangjunwen/nproto/npmsg/durconn"

	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	tststan "github.com/huangjunwen/tstsvc/stan"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/assert"
)

func TestFlush(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestFlush.\n")
	var err error
	assert := assert.New(t)

	var resMySQL *tstmysql.Resource
	{
		resMySQL, err = tstmysql.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer resMySQL.Close()
		log.Printf("MySQL server started.\n")
	}

	var db *sql.DB
	{
		db, err = resMySQL.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	var resStan *tststan.Resource
	{
		resStan, err = tststan.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer resStan.Close()
		log.Printf("Stan server started.\n")
	}

	var nc *nats.Conn
	{
		nc, err = resStan.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Nats client created.\n")
	}

	var dc *durconn.DurConn
	{
		dc, err = durconn.NewDurConn(nc, resStan.ClusterId)
		if err != nil {
			log.Panic(err)
		}
		defer dc.Close()
		log.Printf("DurConn created.\n")
	}

	var store *DBStore
	table := "msgstore"
	{
		store, err = NewDBStore(dc, "mysql", db, table,
			OptMaxInflight(100),
			OptMaxBuf(101),
		)
		assert.Error(err)
		assert.Nil(store)

		store, err = NewDBStore(dc, "mysql", db, table,
			OptMaxInflight(2),
			OptMaxBuf(1),
			OptCreateTable(),
		)
		assert.NoError(err)
		assert.NotNil(store)
		defer store.Close()
		log.Printf("DBStore created.\n")
	}

	// Create a subscription to multiply some distinct prime numbers.
	testSubject := "primeproduct"
	testQueue := "default"
	wg := &sync.WaitGroup{} // wg.Done() is called each time product is updated.
	mu := &sync.Mutex{}
	product := uint64(1)
	resetProduct := func() uint64 {
		mu.Lock()
		ret := product
		product = 1
		mu.Unlock()
		return ret
	}
	{
		c := make(chan struct{})
		dc.Subscribe(
			testSubject,
			testQueue,
			func(ctx context.Context, subject string, data []byte) error {
				// Convert to uint64.
				prime, err := strconv.ParseUint(string(data), 10, 64)
				if err != nil {
					log.Panic(err)
				}

				// Multiply prime and product only when prime has not been multipled.
				// This make the process idempotent: re-delivery the same prime number does not change the product.
				updated := false
				mu.Lock()
				if product%prime != 0 {
					product = product * prime
					updated = true
					log.Printf("** product is updated to %d\n", product)
				}
				mu.Unlock()

				if updated {
					wg.Done()
				}
				return nil
			},
			durconn.SubOptSubscribeCb(func(_ stan.Conn, _, _ string) {
				close(c)
			}),
		)
		<-c
		log.Printf("DurConn subscribed.\n")
	}

	testFlush := func(ctx context.Context, primes []uint64) {
		// Start a transaction.
		tx, err := db.Begin()
		assert.NoError(err)
		defer tx.Rollback()

		// Reset the product.
		resetProduct()

		// Publish distinct prime numbers.
		expect := uint64(1)
		p := store.NewPublisher(tx)
		for _, prime := range primes {
			expect = expect * prime
			wg.Add(1)
			err := p.Publish(ctx, testSubject, []byte(strconv.FormatUint(prime, 10)))
			assert.NoError(err)
		}

		// Commit.
		assert.NoError(tx.Commit())

		// Flush.
		p.Finish(ctx, true)

		// Wait finish.
		wg.Wait()

		// Check.
		assert.Equal(expect, resetProduct())
	}

	testFlush(context.Background(), []uint64{})
	testFlush(context.Background(), []uint64{2})       // flushMsgList
	testFlush(context.Background(), []uint64{3, 5, 7}) // flushMsgStream

}
