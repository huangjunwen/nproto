package durconn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/ory/dockertest"
	docker "github.com/ory/dockertest/docker"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

// ----------- Mock test -----------

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
		testCase{"",
			nil,
			nil,
			false, nil, nil},

		// Reset to nil. Nothing changed.
		testCase{"reset",
			[]interface{}{nil, nil},
			[]interface{}{nil, nil, nil, nil},
			false, nil, nil},

		// Add subscription 1 (without connection).
		testCase{"addSub",
			[]interface{}{sub1},
			[]interface{}{nil, nil, nil}, // No current connection, the real subscription is defer to later connection reset. See next test case.
			false, nil, []*subscription{sub1}},

		// Reset to connection 1.
		testCase{"reset",
			[]interface{}{sc1, stalec1},
			[]interface{}{nil, nil, []*subscription{sub1}, nil},
			false, sc1, []*subscription{sub1}},

		// Add subscription 2 (with connection).
		testCase{"addSub",
			[]interface{}{sub2},
			[]interface{}{sc1, stalec1, nil}, // Has connection, the real subscription should be run immediately.
			false, sc1, []*subscription{sub1, sub2}},

		// Add subscription 1 again (with connection) should result an error e.g. duplicated.
		testCase{"addSub",
			[]interface{}{sub1},
			[]interface{}{nil, nil, ErrDupSubscription(sub1.subject, sub1.queue)},
			false, sc1, []*subscription{sub1, sub2}},

		// Reset to connection 2.
		testCase{"reset",
			[]interface{}{sc2, stalec2},
			[]interface{}{sc1, stalec1, []*subscription{sub1, sub2}, nil}, // sc1 should be released and sub1/sub2 should be re-subscribed.
			false, sc2, []*subscription{sub1, sub2}},

		// Now close.
		testCase{"close",
			nil,
			[]interface{}{sc2, stalec2, nil}, // sc2 should be released.
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

// ----------- Real test -----------

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

func natsURL() string {
	return "nats://localhost:42222"
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
		PortBindings: map[docker.Port][]docker.PortBinding{
			"4222/tcp": []docker.PortBinding{
				docker.PortBinding{
					HostIP:   "127.0.0.1",
					HostPort: "42222",
				},
			},
		},
	}

	// If host's dataDir is not empty, then mount to it.
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
			stan.NatsURL(natsURL()))
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

// TestConnect tests connects and reconnects.
func TestConnect(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestConnect.\n")
	var err error

	// Starts the first server.
	var res1 *dockertest.Resource
	{
		log.Printf("Starting streaming server.\n")
		res1, err = runTestServer("")
		if err != nil {
			log.Fatal(err)
		}
		defer res1.Close()
		log.Printf("Streaming server started.\n")
	}

	// Creates nats connection (with infinite reconnection).
	var nc *nats.Conn
	{
		nc, err = nats.Connect(natsURL(),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Close()
		log.Printf("Nats connection created.\n")
	}

	// Creates DurConn.
	var dc *DurConn
	connectc := make(chan struct{})
	disconnectc := make(chan struct{})
	{
		connectCb := func(_ stan.Conn) { connectc <- struct{}{} }
		disconnectCb := func(_ stan.Conn) { disconnectc <- struct{}{} }

		dc, err = NewDurConn(nc, stanClusterId,
			DurConnOptConnectCb(connectCb),       // Use the callback to notify connection establish.
			DurConnOptDisconnectCb(disconnectCb), // Use the callback to notify disconnection.
			DurConnOptReconnectWait(time.Second), // A short reconnect wait.
			DurConnOptPings(1, 3),                // Minimal pings.
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

	// First server gone.
	res1.Close()
	log.Printf("Streaming server closed.\n")

	// Wait disconnection.
	<-disconnectc
	log.Printf("DurConn disconnected.\n")

	// Starts the second server.
	var res2 *dockertest.Resource
	{
		log.Printf("Starting another streaming server.\n")
		res2, err = runTestServer("")
		if err != nil {
			log.Fatal(err)
		}
		defer res2.Close()
		log.Printf("Another Streaming server started.\n")
	}

	// Wait connection.
	<-connectc
	log.Printf("DurConn connected again.\n")
}

// TestPubSub tests message publishing and subscribing.
func TestPubSub(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestPubSub.\n")
	assert := assert.New(t)
	var err error

	// Use a tmp directory as data dir.
	var datadir string
	{
		datadir, err = ioutil.TempDir("/tmp", "durconn_test")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(datadir)
		log.Printf("Temp data dir created: %s.\n", datadir)
	}

	// Starts the first server.
	var res1 *dockertest.Resource
	{
		log.Printf("Starting streaming server.\n")
		res1, err = runTestServer(datadir)
		if err != nil {
			log.Fatal(err)
		}
		defer res1.Close()
		log.Printf("Streaming server started.\n")
	}

	// Creates nats connection (with infinite reconnection).
	var nc *nats.Conn
	{
		nc, err = nats.Connect(natsURL(),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Close()
		log.Printf("Nats connection created.\n")
	}

	// Creates DurConn.
	var dc *DurConn
	connectc := make(chan struct{})
	disconnectc := make(chan struct{})
	{
		connectCb := func(_ stan.Conn) { connectc <- struct{}{} }
		disconnectCb := func(_ stan.Conn) { disconnectc <- struct{}{} }

		dc, err = NewDurConn(nc, stanClusterId,
			DurConnOptConnectCb(connectCb),       // Use the callback to notify connection establish.
			DurConnOptDisconnectCb(disconnectCb), // Use the callback to notify disconnection.
			DurConnOptReconnectWait(time.Second), // A short reconnect wait.
			DurConnOptPings(1, 3),                // Minimal pings.
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

	// Abnormal subscription.
	{
		// NOTE: nats streaming does not support wildcard.
		// This will make this subscription loop infinitely.
		err := dc.Subscribe(
			"sub.*",
			"q1",
			func(_ context.Context, _ string, _ []byte) error { return nil },
			SubOptRetryWait(time.Second),
		)
		assert.NoError(err)
	}

	// Normal subscription.
	var (
		testSubject = "sub.sub"
		testQueue   = "qq"
		goodData    = []byte("good")
		badData     = []byte("bad")
	)

	subc := make(chan struct{})
	goodc := make(chan struct{}, 3)
	badc := make(chan struct{}, 3)
	{

		subCb := func(_ stan.Conn, _, _ string) { subc <- struct{}{} }
		err := dc.Subscribe(
			testSubject,
			testQueue,
			func(ctx context.Context, subject string, data []byte) error {
				log.Printf("*** Handler called %+q %+q.\n", subject, data)
				assert.Equal(testSubject, subject)
				if bytes.Equal(data, goodData) {
					goodc <- struct{}{}
					return nil
				} else {
					// NOTE: since returnning error will cause message redelivery.
					// Don't block here.
					select {
					case badc <- struct{}{}:
					default:
					}
					return errors.New("bad data")
				}
			},
			SubOptSubscribeCb(subCb),
			SubOptAckWait(time.Second), // Short ack wait results in fast redelivery.
		)
		assert.NoError(err)

		<-subc
		log.Printf("Subscribed: %v.\n", err)
	}

	// Publish good data.
	{
		err := dc.Publish(context.Background(),
			testSubject,
			goodData,
		)
		assert.NoError(err)
		<-goodc
		log.Printf("Publish good data: %v.\n", err)
	}

	// PublishBatch good data.
	{
		errs := dc.PublishBatch(context.Background(),
			[]string{testSubject, testSubject},
			[][]byte{goodData, goodData},
		)
		for _, err := range errs {
			assert.NoError(err)
		}
		<-goodc
		<-goodc
		log.Printf("PublishBatch good data: %v.\n", errs)
	}

	// Publish bad data.
	{
		err := dc.Publish(context.Background(),
			testSubject,
			badData,
		)
		assert.NoError(err)
		<-badc
		log.Printf("Publish bad data: %v.\n", err)
	}

	// First server gone.
	res1.Close()
	log.Printf("Streaming server closed.\n")

	// Wait disconnection.
	<-disconnectc
	log.Printf("DurConn disconnected.\n")

	// Starts the second server.
	var res2 *dockertest.Resource
	{
		log.Printf("Starting another streaming server.\n")
		res2, err = runTestServer(datadir)
		if err != nil {
			log.Fatal(err)
		}
		defer res2.Close()
		log.Printf("Another Streaming server started.\n")
	}

	// Wait re connection.
	<-connectc
	log.Printf("DurConn connected again.\n")

	// Wait re subscription.
	<-subc
	log.Printf("Subscribed again.\n")

	// Wait bad data redelivery.
	<-badc
	<-badc
	log.Printf("Bad data redelivery.\n")
}
