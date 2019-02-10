package dbstore

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	tststan "github.com/huangjunwen/tstsvc/stan"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/nproto/npmsg"
	"github.com/huangjunwen/nproto/nproto/npmsg/durconn"
)

// UnstableAsyncPublisher makes half publish failed.
type UnstableAsyncPublisher struct {
	npmsg.RawMsgAsyncPublisher
	cnt int64
}

// SyncPublisher shadows PublishAsync method.
type SyncPublisher struct {
	publisher npmsg.RawMsgAsyncPublisher
}

var (
	_           npmsg.RawMsgAsyncPublisher = (*UnstableAsyncPublisher)(nil)
	_           npmsg.RawMsgPublisher      = (*SyncPublisher)(nil)
	errUnstable error                      = errors.New("Unstable error")
)

func NewUnstableAsyncPublisher(p npmsg.RawMsgAsyncPublisher) *UnstableAsyncPublisher {
	return &UnstableAsyncPublisher{
		RawMsgAsyncPublisher: p,
	}
}

func (p *UnstableAsyncPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	return npmsg.RawMsgAsyncPublisherFunc(p.PublishAsync).Publish(ctx, subject, data)
}

func (p *UnstableAsyncPublisher) PublishAsync(ctx context.Context, subject string, data []byte, cb func(error)) error {
	cnt := atomic.AddInt64(&p.cnt, 1)
	// If cnt is even, then failed.
	if cnt%2 == 0 {
		if cnt == 2 {
			// Failed directly.
			return errUnstable
		} else {
			// Failed after some time.
			time.AfterFunc(time.Duration(cnt)*time.Millisecond, func() {
				cb(errUnstable)
			})
			return nil
		}
	}
	return p.RawMsgAsyncPublisher.PublishAsync(ctx, subject, data, cb)
}

func NewSyncPublisher(p npmsg.RawMsgAsyncPublisher) *SyncPublisher {
	return &SyncPublisher{
		publisher: p,
	}
}

func (p *SyncPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	return p.publisher.Publish(ctx, subject, data)
}

func TestFlush(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestFlush.\n")
	var err error
	assert := assert.New(t)

	bgctx := context.Background()

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

	// Starts test stan server.
	var resStan *tststan.Resource
	{
		resStan, err = tststan.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer resStan.Close()
		log.Printf("Stan server started.\n")
	}

	// Connects to embedded nats server.
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

	// Creates DurConn.
	var dc *durconn.DurConn
	{
		dc, err = durconn.NewDurConn(nc, resStan.ClusterId)
		if err != nil {
			log.Panic(err)
		}
		defer dc.Close()
		log.Printf("DurConn created.\n")
	}

	// Create a subscription to multiply some DISTINCT prime numbers.
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

				// Multiply prime and product only when prime has not been multiplied.
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

	// Helper functions.
	table := "msgstore"
	createStore := func(downstream npmsg.RawMsgPublisher) *DBStore {
		// Creates DBStore with small MaxInflight/MaxBuf/FlushWait.
		store, err := NewDBStore(downstream, "mysql", db, table,
			OptMaxInflight(3),
			OptMaxBuf(2),
			OptCreateTable(),
			OptFlushWait(500*time.Millisecond), // Short flush wait.
			OptNoRedeliveryLoop(),              // Don't run the delivery loop.
		)
		if err != nil {
			log.Panic(err)
		}
		return store
	}
	clearMsgTable := func() {
		_, err := db.Exec("DELETE FROM " + table)
		assert.NoError(err)
	}
	assertMsgTableRows := func(expect int) {
		cnt := 0
		assert.NoError(db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&cnt))
		assert.Equal(expect, cnt)
	}

	// --- Test normal case ---
	log.Printf(">>> Test normal cases...\n")
	testNormalFlush := func(primes []uint64, async bool) {
		log.Printf("Begin normal case: %v, %v\n", primes, async)
		// Create store.
		var store *DBStore
		if async {
			store = createStore(dc)
		} else {
			store = createStore(NewSyncPublisher(dc))
		}
		defer store.Close()

		// Make sure msg table is empty.
		clearMsgTable()
		defer clearMsgTable()

		// Make sure product reset.
		resetProduct()
		defer resetProduct()

		// Start a transaction.
		tx, err := db.Begin()
		assert.NoError(err)
		defer tx.Rollback()

		// Creates a publisher.
		p := store.NewPublisher(tx)

		// Publish distinct prime numbers.
		expect := uint64(1)
		for _, prime := range primes {
			err := p.Publish(bgctx, testSubject, []byte(strconv.FormatUint(prime, 10)))
			assert.NoError(err)
			expect = expect * prime
		}

		// Commit.
		assert.NoError(tx.Commit())

		// Check database rows.
		assertMsgTableRows(len(primes))

		// Flush.
		wg.Add(len(primes))
		p.Flush(bgctx)
		wg.Wait()

		// Check database rows.
		assertMsgTableRows(0)

		// Check.
		assert.Equal(expect, resetProduct())

		log.Printf("End normal case: %v, %v\n", primes, async)
	}

	testNormalFlush([]uint64{}, true)
	testNormalFlush([]uint64{}, false)
	testNormalFlush([]uint64{2, 3}, true)              // flushMsgList
	testNormalFlush([]uint64{2, 3}, false)             // flushMsgList
	testNormalFlush([]uint64{5, 7, 11, 13, 17}, true)  // flushMsgStream
	testNormalFlush([]uint64{5, 7, 11, 13, 17}, false) // flushMsgStream

	// --- Test error case ---
	log.Printf(">>> Test error cases...\n")
	testErrorFlush := func(primes []uint64, async bool) {
		log.Printf("Begin error case: %v, %v\n", primes, async)
		// Create store.
		var store *DBStore
		if async {
			store = createStore(NewUnstableAsyncPublisher(dc))
		} else {
			store = createStore(NewSyncPublisher(NewUnstableAsyncPublisher(dc)))
		}
		defer store.Close()

		// Make sure msg table is empty.
		clearMsgTable()
		defer clearMsgTable()

		// Make sure product reset.
		resetProduct()
		defer resetProduct()

		// Start a transaction.
		tx, err := db.Begin()
		assert.NoError(err)
		defer tx.Rollback()

		// Creates a publisher.
		p := store.NewPublisher(tx)

		// Publish distinct prime numbers.
		for _, prime := range primes {
			err := p.Publish(bgctx, testSubject, []byte(strconv.FormatUint(prime, 10)))
			assert.NoError(err)
		}

		// Commit.
		assert.NoError(tx.Commit())

		// Check database rows.
		assertMsgTableRows(len(primes))

		// UnstableAsyncPublisher makes publishing half failed.
		expectSucc := len(primes)/2 + len(primes)%2

		// Flush.
		wg.Add(expectSucc)
		p.Flush(bgctx)
		wg.Wait()

		// Check database rows.
		assertMsgTableRows(len(primes) - expectSucc)

		log.Printf("End normal case: %v, %v\n", primes, async)
	}

	testErrorFlush([]uint64{}, true)
	testErrorFlush([]uint64{}, false)
	testErrorFlush([]uint64{2, 3}, true)              // flushMsgList
	testErrorFlush([]uint64{2, 3}, false)             // flushMsgList
	testErrorFlush([]uint64{5, 7, 11, 13, 17}, true)  // flushMsgStream
	testErrorFlush([]uint64{5, 7, 11, 13, 17}, false) // flushMsgStream

	// --- Test redelivery flush ---
	log.Printf(">>> Test redelivery ...\n")

	testRedelivery := func(primes []uint64, async bool) {
		log.Printf("Begin redelivery: %v, %v\n", primes, async)
		// Create store.
		var store *DBStore
		if async {
			store = createStore(dc)
		} else {
			store = createStore(NewSyncPublisher(dc))
		}
		store.redeliveryLoop()
		defer store.Close()

		// Make sure msg table is empty.
		clearMsgTable()
		defer clearMsgTable()

		// Make sure product reset.
		resetProduct()
		defer resetProduct()

		// Start a transaction.
		tx, err := db.Begin()
		assert.NoError(err)
		defer tx.Rollback()

		// Creates a publisher.
		p := store.NewPublisher(tx)

		// Publish distinct prime numbers.
		expect := uint64(1)
		for _, prime := range primes {
			err := p.Publish(bgctx, testSubject, []byte(strconv.FormatUint(prime, 10)))
			assert.NoError(err)
			expect = expect * prime
		}

		// NOTE: Add wait group before commit, since once committed, the redeliveryLoop run immediately.
		wg.Add(len(primes))

		// Commit.
		assert.NoError(tx.Commit())

		// NOTE: Not call p.Finish, let redeliveryLoop to do it.
		wg.Wait()

		// Check.
		assert.Equal(expect, resetProduct())

		log.Printf("End redelivery: %v, %v\n", primes, async)
	}

	testRedelivery([]uint64{}, true)
	testRedelivery([]uint64{}, false)
	testRedelivery([]uint64{2, 3}, true)
	testRedelivery([]uint64{2, 3}, false)
	testRedelivery([]uint64{5, 7, 11, 13, 17}, true)
	testRedelivery([]uint64{5, 7, 11, 13, 17}, false)
}
