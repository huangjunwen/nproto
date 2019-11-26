package dbpipe

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
	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/helpers/sql"
	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/npmsg/durconn"
)

// UnstableAsyncPublisher makes half publish failed.
type UnstableAsyncPublisher struct {
	publisher nproto.MsgAsyncPublisher
	cnt       int64
}

// SyncPublisher shadows PublishAsync method.
type SyncPublisher struct {
	publisher nproto.MsgAsyncPublisher
}

var (
	_           nproto.MsgAsyncPublisher = (*UnstableAsyncPublisher)(nil)
	_           nproto.MsgPublisher      = (*SyncPublisher)(nil)
	errUnstable error                    = errors.New("Unstable error")
)

func NewUnstableAsyncPublisher(p nproto.MsgAsyncPublisher) *UnstableAsyncPublisher {
	return &UnstableAsyncPublisher{
		publisher: p,
	}
}

func (p *UnstableAsyncPublisher) Publish(ctx context.Context, subject string, msgData []byte) error {
	return nproto.MsgAsyncPublisherFunc(p.PublishAsync).Publish(ctx, subject, msgData)
}

func (p *UnstableAsyncPublisher) PublishAsync(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
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
	return p.publisher.PublishAsync(ctx, subject, msgData, cb)
}

func NewSyncPublisher(p nproto.MsgAsyncPublisher) *SyncPublisher {
	return &SyncPublisher{
		publisher: p,
	}
}

func (p *SyncPublisher) Publish(ctx context.Context, subject string, msgData []byte) error {
	return nproto.MsgAsyncPublisherFunc(p.publisher.PublishAsync).Publish(ctx, subject, msgData)
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
		subc := make(chan struct{})
		dc.Subscribe(
			testSubject,
			testQueue,
			func(ctx context.Context, msgData []byte) error {
				// Convert to uint64.
				prime, err := strconv.ParseUint(string(msgData), 10, 64)
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
				close(subc)
			}),
		)
		<-subc
		log.Printf("DurConn subscribed.\n")
	}

	// Helper functions.
	table := "msgstore"
	createPipe := func(downstream nproto.MsgPublisher) *DBMsgPublisherPipe {
		// Creates DBMsgPublisherPipe with small MaxInflight/MaxBuf/FlushWait.
		pipe, err := NewDBMsgPublisherPipe(downstream, "mysql", db, table,
			OptMaxInflight(3),
			OptMaxBuf(2),
			OptFlushWait(500*time.Millisecond), // Short flush wait.
			OptNoRedeliveryLoop(),              // Don't run the delivery loop.
		)
		if err != nil {
			log.Panic(err)
		}
		return pipe
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
		// Create pipe.
		var pipe *DBMsgPublisherPipe
		if async {
			pipe = createPipe(dc)
		} else {
			pipe = createPipe(NewSyncPublisher(dc))
		}
		defer pipe.Close()

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
		p := pipe.NewMsgPublisher(tx)

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
		// Create pipe.
		var pipe *DBMsgPublisherPipe
		if async {
			pipe = createPipe(NewUnstableAsyncPublisher(dc))
		} else {
			pipe = createPipe(NewSyncPublisher(NewUnstableAsyncPublisher(dc)))
		}
		defer pipe.Close()

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
		p := pipe.NewMsgPublisher(tx)

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
	testErrorFlush([]uint64{3, 7}, true)              // flushMsgList
	testErrorFlush([]uint64{3, 7}, false)             // flushMsgList
	testErrorFlush([]uint64{2, 7, 11, 13, 17}, true)  // flushMsgStream
	testErrorFlush([]uint64{2, 7, 11, 13, 17}, false) // flushMsgStream

	// --- Test redelivery flush ---
	log.Printf(">>> Test redelivery ...\n")

	testRedelivery := func(primes []uint64, async bool) {
		log.Printf("Begin redelivery: %v, %v\n", primes, async)
		// Create pipe.
		var pipe *DBMsgPublisherPipe
		if async {
			pipe = createPipe(dc)
		} else {
			pipe = createPipe(NewSyncPublisher(dc))
		}
		pipe.redeliveryLoop()
		defer pipe.Close()

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
		p := pipe.NewMsgPublisher(tx)

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

		// NOTE: Not call p.Flush, let redeliveryLoop to do it.
		wg.Wait()

		// Check.
		assert.Equal(expect, resetProduct())

		log.Printf("End redelivery: %v, %v\n", primes, async)
	}

	testRedelivery([]uint64{}, true)
	testRedelivery([]uint64{}, false)
	testRedelivery([]uint64{11, 13}, true)
	testRedelivery([]uint64{11, 13}, false)
	testRedelivery([]uint64{2, 7, 11, 13, 3}, true)
	testRedelivery([]uint64{2, 7, 11, 13, 3}, false)

	// --- Test WithTx ---
	log.Printf(">>> Test WithTx ...\n")
	testWithTx := func(primes []uint64, async bool) {
		log.Printf("Begin WithTx: %v, %v\n", primes, async)
		// Create pipe.
		var pipe *DBMsgPublisherPipe
		if async {
			pipe = createPipe(dc)
		} else {
			pipe = createPipe(NewSyncPublisher(dc))
		}
		defer pipe.Close()

		// Make sure msg table is empty.
		clearMsgTable()
		defer clearMsgTable()

		// Make sure product reset.
		resetProduct()
		defer resetProduct()

		expect := uint64(1)
		assert.NoError(sqlh.WithTx(bgctx, db, func(ctx context.Context, tx *sql.Tx) error {
			// Creates a publisher.
			p := pipe.NewMsgPublisherWithTx(ctx, tx)

			// Publish distinct prime numbers.
			for _, prime := range primes {
				err := p.Publish(ctx, testSubject, []byte(strconv.FormatUint(prime, 10)))
				assert.NoError(err)
				expect = expect * prime
			}

			// NOTE: Add wait group before commit
			wg.Add(len(primes))

			// return nil to commit
			return nil
		}))

		wg.Wait()

		// Check.
		assert.Equal(expect, resetProduct())

		log.Printf("End WithTx: %v, %v\n", primes, async)
	}

	testWithTx([]uint64{}, true)
	testWithTx([]uint64{}, false)
	testWithTx([]uint64{7, 13}, true)            // flushMsgList
	testWithTx([]uint64{7, 13}, false)           // flushMsgList
	testWithTx([]uint64{5, 2, 11, 13, 7}, true)  // flushMsgStream
	testWithTx([]uint64{5, 2, 11, 13, 7}, false) // flushMsgStream
}
