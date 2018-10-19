package durconn

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/ory/dockertest"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// ----------- Mock begins -----------

type MockConn int

var (
	_ stan.Conn = MockConn(0)
)

func newMockConn(i int) MockConn {
	return MockConn(i)
}

func (mc MockConn) Publish(subject string, data []byte) error {
	panic("Not implemented")
}

func (mc MockConn) PublishAsync(subject string, data []byte, ah stan.AckHandler) (string, error) {
	panic("Not implemented")
}

func (mc MockConn) Subscribe(subject string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	panic("Not implemented")
}

func (mc MockConn) QueueSubscribe(subject, queue string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	panic("Not implemented")
}

func (mc MockConn) Close() error {
	panic("Not implemented")
}

func (mc MockConn) NatsConn() *nats.Conn {
	panic("Not implemented")
}

// ----------- Mock ends -----------

// Test reset/addSub/close_ .
// They are the 'atomic' methods of DurConn since they all use:
//
//   dc.mu.Lock()
//   defer dc.mu.Unlock()
//
// for the whole function.
func TestLockMethods(t *testing.T) {
	assert := assert.New(t)

	dc := &DurConn{
		subNames: map[[2]string]int{},
	}
	sc1 := newMockConn(1)
	stalec1 := make(chan struct{})
	sc2 := newMockConn(2)
	stalec2 := make(chan struct{})
	sub1 := &subscription{
		subject: "subject",
		queue:   "good1",
	}
	sub2 := &subscription{
		subject: "subject",
		queue:   "good2",
	}
	sub3 := &subscription{
		subject: "subject",
		queue:   "good3",
	}

	type testCase struct {
		Method       string
		Args         []interface{}
		ExpectResult []interface{}
		Closed       bool
		Sc           stan.Conn
		Subs         []*subscription
	}

	runTestCase := func(cases []testCase) {
		for i, c := range cases {
			i += 1
			args := c.Args
			result := c.ExpectResult

			switch c.Method {
			case "reset":
				var (
					sc     stan.Conn
					stalec chan struct{}
				)
				if args[0] != nil {
					sc = args[0].(stan.Conn)
				}
				if args[1] != nil {
					stalec = args[1].(chan struct{})
				}

				oldSc, oldStalec, subs, err := dc.reset(sc, stalec)

				if result[0] == nil {
					assert.Nil(oldSc, "#%d", i)
				} else {
					assert.Equal(result[0], oldSc, "#%d", i)
				}
				if result[1] == nil {
					assert.Nil(oldStalec, "#%d", i)
				} else {
					assert.Equal(result[1], oldStalec, "#%d", i)
				}
				if result[2] == nil {
					assert.Nil(subs, "#%d", i)
				} else {
					assert.Equal(result[2], subs, "#%d", i)
				}
				if result[3] == nil {
					assert.Nil(err, "#%d", i)
				} else {
					assert.Equal(result[3], err, "#%d", i)
				}

			case "addSub":
				var (
					sub *subscription
				)
				if args[0] != nil {
					sub = args[0].(*subscription)
				}

				sc, stalec, err := dc.addSub(sub)

				if result[0] == nil {
					assert.Nil(sc, "#%d", i)
				} else {
					assert.Equal(result[0], sc, "#%d", i)
				}
				if result[1] == nil {
					assert.Nil(stalec, "#%d", i)
				} else {
					assert.Equal(result[1], stalec, "#%d", i)
				}
				if result[2] == nil {
					assert.Nil(err, "#%d", i)
				} else {
					assert.Equal(result[2], err, "#%d", i)
				}

			case "close":
				oldSc, oldStalec, err := dc.close_()

				if result[0] == nil {
					assert.Nil(oldSc, "#%d", i)
				} else {
					assert.Equal(result[0], oldSc, "#%d", i)
				}
				if result[1] == nil {
					assert.Nil(oldStalec, "#%d", i)
				} else {
					assert.Equal(result[1], oldStalec, "#%d", i)
				}
				if result[2] == nil {
					assert.Nil(err, "#%d", i)
				} else {
					assert.Equal(result[2], err, "#%d", i)
				}

			default:
			}

			assert.Equal(c.Closed, dc.closed, "#%d", i)
			assert.Equal(c.Sc, dc.sc, "#%d", i)
			assert.Equal(c.Subs, dc.subs, "#%d", i)
		}
	}

	runTestCase([]testCase{
		// Init state.
		testCase{"", nil, nil, false, nil, nil},
		// Reset to nil. Nothing changed.
		testCase{"reset",
			[]interface{}{nil, nil},
			[]interface{}{nil, nil, nil, nil},
			false, nil, nil},
		// Add subscription 1 (without connection).
		testCase{"addSub",
			[]interface{}{sub1},
			[]interface{}{nil, nil, nil},
			false, nil, []*subscription{sub1}},
		// Reset to connection 1.
		testCase{"reset",
			[]interface{}{sc1, stalec1},
			[]interface{}{nil, nil, []*subscription{sub1}, nil},
			false, sc1, []*subscription{sub1}},
		// Add subscription 2 (with connection).
		testCase{"addSub",
			[]interface{}{sub2},
			[]interface{}{sc1, stalec1, nil},
			false, sc1, []*subscription{sub1, sub2}},
		// Add subscription 1 again (with connection) should result an error.
		testCase{"addSub",
			[]interface{}{sub1},
			[]interface{}{nil, nil, ErrDupSubscription(sub1.subject, sub1.queue)},
			false, sc1, []*subscription{sub1, sub2}},
		// Reset to connection 2.
		testCase{"reset",
			[]interface{}{sc2, stalec2},
			[]interface{}{sc1, stalec1, []*subscription{sub1, sub2}, nil},
			false, sc2, []*subscription{sub1, sub2}},
		// Now close.
		testCase{"close",
			nil,
			[]interface{}{sc2, stalec2, nil},
			true, nil, []*subscription{sub1, sub2}},
		// Further calls should result ErrClosed.
		testCase{"reset",
			[]interface{}{sc1, stalec1},
			[]interface{}{nil, nil, nil, ErrClosed},
			true, nil, []*subscription{sub1, sub2}},
		testCase{"addSub",
			[]interface{}{sub3},
			[]interface{}{nil, nil, ErrClosed},
			true, nil, []*subscription{sub1, sub2}},
		testCase{"close",
			nil,
			[]interface{}{nil, nil, ErrClosed},
			true, nil, []*subscription{sub1, sub2}},
	})

}

const (
	stanClusterId = "durconn_test"
	stanTag       = "0.11.0-linux"
)

var (
	pool *dockertest.Pool
)

func init() {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatal(err)
	}
}

func natsURL(res *dockertest.Resource) string {
	return fmt.Sprintf("nats://localhost:%s", res.GetPort("4222/tcp"))
}

func runTestServer(dataDir string) (*dockertest.Resource, error) {
	// Use file store.
	opts := &dockertest.RunOptions{
		Repository: "nats-streaming",
		Tag:        stanTag,
		Cmd: []string{
			"-cid", stanClusterId,
			"-st", "FILE",
			"--dir", "/data",
		},
	}
	if dataDir != "" {
		opts.Mounts = []string{fmt.Sprintf("%s:/data", dataDir)}
	}

	// Start the container.
	res, err := pool.RunWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// Set max lifetime of the container.
	res.Expire(120)

	// Wait.
	if err := pool.Retry(func() error {
		sc, err := stan.Connect(stanClusterId, xid.New().String(),
			stan.NatsURL(natsURL(res)))
		if err != nil {
			return err
		}
		sc.Close()
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func TestBasicPubSub(t *testing.T) {
	log.Printf(">>> TestBasicPubSub.\n")
	assert := assert.New(t)
	var err error

	var res *dockertest.Resource
	{
		log.Printf("Starting streaming server.\n")
		res, err = runTestServer("")
		if err != nil {
			log.Fatal(err)
		}
		defer res.Close()
		log.Printf("Streaming server started.\n")
	}

	var nc *nats.Conn
	{
		nc, err = nats.Connect(natsURL(res),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Close()
		log.Printf("Nats connection created.\n")
	}

	var dc *DurConn
	{
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		connectc := make(chan struct{})
		connectCb := func(_ stan.Conn) { connectc <- struct{}{} }

		dc, err = NewDurConn(nc, stanClusterId,
			DurConnOptLogger(&logger),
			DurConnOptConnectCb(connectCb), // Use the callback to notify connection establish.
		)
		if err != nil {
			log.Fatal(err)
		}
		defer dc.Close()
		log.Printf("DurConn created.\n")

		// Wait connection.
		<-connectc
		log.Printf("DurConn connected.\n")
	}

	const (
		testSubject = "subsub"
		testQueue   = "qq"
		testData    = "datadata"
	)

	wg := &sync.WaitGroup{}

	{
		subc := make(chan struct{})
		subCb := func(_ stan.Conn, _, _ string) { subc <- struct{}{} }

		err := dc.Subscribe(
			testSubject,
			testQueue,
			func(ctx context.Context, subject string, data []byte) error {
				log.Printf("Handler called\n")
				assert.Equal(testSubject, subject)
				assert.Equal(testData, string(data))
				wg.Done()
				return nil
			},
			SubOptSubscribeCb(subCb),
		)
		assert.NoError(err)

		<-subc
		log.Printf("Subscribed: %v\n", err)
	}

	{
		wg.Add(1)
		err := dc.Publish(context.Background(), testSubject, []byte(testData))
		log.Printf("Published: %v\n", err)
		assert.NoError(err)
	}

	wg.Wait()

}
