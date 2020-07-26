package stanmsg

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/huangjunwen/golibs/taskrunner/limitedrunner"
	"github.com/huangjunwen/tstsvc"
	tststan "github.com/huangjunwen/tstsvc/stan"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
)

func newLogger() logr.Logger {
	out := zerolog.NewConsoleWriter()
	out.TimeFormat = time.RFC3339
	out.Out = os.Stderr
	lg := zerolog.New(&out).With().Timestamp().Logger()
	return (*zerologr.Logger)(&lg)
}

func TestConnect(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestConnect.\n")
	var err error

	opts := &tststan.Options{
		HostPort: tstsvc.FreePort(),
	}

	// Starts the first server.
	var res1 *tststan.Resource
	{
		res1, err = tststan.Run(opts)
		if err != nil {
			log.Panic(err)
		}
		defer res1.Close()
		log.Printf("Test stan server 1 started: %+q\n", res1.NatsURL())
	}

	// Creates nats connection (with infinite reconnection).
	var nc *nats.Conn
	{
		nc, err = res1.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Connection connected.\n")

		if false {
			// Display raw nats messages flow.
			nc.Subscribe(">", func(msg *nats.Msg) {
				log.Printf("***** subject=%s reply=%s data=(hex)%x len=%d\n", msg.Subject, msg.Reply, msg.Data, len(msg.Data))
			})
		}
	}

	// Creates DurConn.
	var dc *DurConn
	connectC := make(chan stan.Conn, 1)
	disconnectC := make(chan stan.Conn, 1)
	{
		dc, err = NewDurConn(
			nc,
			res1.Options.ClusterId,
			DCOptLogger(newLogger()),
			DCOptReconnectWait(time.Second), // short reconnect wait.
			DCOptStanPingInterval(1),        // min ping interval.
			DCOptStanPingMaxOut(2),          // min ping max out.
			DCOptConnectCb(func(sc stan.Conn) { connectC <- sc }),
			DCOptDisconnectCb(func(sc stan.Conn) { disconnectC <- sc }),
		)
		if err != nil {
			log.Panic(err)
		}
		defer dc.Close()
		log.Printf("DurConn created.\n")
	}

	// Wait connect.
	log.Printf("DurConn connected %p.\n", <-connectC)

	// First server gone.
	res1.Close()
	log.Printf("Test stan server 1 closed\n")

	// Wait disconnect.
	log.Printf("DurConn disconnected %p.\n", <-disconnectC)

	// Wait a while to see reconnect loop.
	time.Sleep(3 * time.Second)

	// Starts the second server using same options.
	var res2 *tststan.Resource
	{
		res2, err = tststan.Run(opts)
		if err != nil {
			log.Panic(err)
		}
		defer res2.Close()
		log.Printf("Test stan server 2 started: %+q\n", res2.NatsURL())
	}

	// Wait reconnect.
	log.Printf("DurConn reconnect %p.\n", <-connectC)
}

func TestPubSub(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestPubSub.\n")
	var err error
	assert := assert.New(t)

	// Use a tmp directory as data dir.
	var datadir string
	{
		datadir, err = ioutil.TempDir("/tmp", "durconn_test")
		if err != nil {
			log.Panic(err)
		}
		defer os.RemoveAll(datadir)
		log.Printf("Temp data dir created: %s.\n", datadir)
	}

	opts := &tststan.Options{
		FileStore:    true,
		HostDataPath: datadir,
		HostPort:     tstsvc.FreePort(),
	}

	// Starts the first server.
	var res1 *tststan.Resource
	{
		res1, err = tststan.Run(opts)
		if err != nil {
			log.Panic(err)
		}
		defer res1.Close()
		log.Printf("Test stan server 1 started: %+q\n", res1.NatsURL())
	}

	// Creates nats connection (with infinite reconnection).
	var nc *nats.Conn
	{
		nc, err = res1.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Connection connected.\n")

		if false {
			// Display raw nats messages flow.
			nc.Subscribe(">", func(msg *nats.Msg) {
				log.Printf("***** subject=%s reply=%s data=(hex)%x len=%d\n", msg.Subject, msg.Reply, msg.Data, len(msg.Data))
			})
		}
	}

	// Values for checks.
	type CtxKey struct{}
	ctxVal := "123"
	ctx := context.WithValue(context.Background(), CtxKey{}, ctxVal)

	mdKey := "mdKey"
	mdVal := "mdVal"

	spec := MustMsgSpec(
		"app.topic",
		func() interface{} { return wrapperspb.String("") },
	)
	spec2 := MustMsgSpec(
		"app.topic",
		func() interface{} { return new(string) },
	)
	specWrongType := MustMsgSpec(
		"app.topic",
		func() interface{} { return &map[string]interface{}{} },
	)
	queue := "default"

	goodData := "good"
	badData := "bad"

	// Creates DurConn.
	var dc *DurConn
	connectC := make(chan stan.Conn, 1)
	disconnectC := make(chan stan.Conn, 1)
	subC := make(chan MsgSpec, 1)
	{
		dc, err = NewDurConn(
			nc,
			res1.Options.ClusterId,
			DCOptLogger(newLogger()),
			DCOptRunner(limitedrunner.Must()),
			DCOptSubjectPrefix("stanmsg2"),
			DCOptContext(ctx),
			DCOptReconnectWait(time.Second), // short reconnect wait.
			DCOptStanPingInterval(1),        // min ping interval.
			DCOptStanPingMaxOut(2),          // min ping max out.
			DCOptConnectCb(func(sc stan.Conn) { connectC <- sc }),
			DCOptDisconnectCb(func(sc stan.Conn) { disconnectC <- sc }),
			DCOptSubscribeCb(func(_ stan.Conn, spec MsgSpec) { subC <- spec }),
		)
		if err != nil {
			log.Panic(err)
		}
		defer dc.Close()
		log.Printf("DurConn created.\n")
	}

	// Wait connect.
	log.Printf("DurConn connected %p.\n", <-connectC)

	publisher := PbJsonPublisher(dc)
	subscriber := PbJsonSubscriber(dc)

	// Subscribe.
	goodC := make(chan string, 10)
	badC := make(chan string, 10)
	{
		err = subscriber.Subscribe(
			spec,
			queue,
			func(ctx context.Context, msg interface{}) error {
				// Check base context.
				assert.Equal(ctxVal, ctx.Value(CtxKey{}))

				// Check md passing.
				md := npmd.MDFromIncomingContext(ctx)
				assert.Equal(mdVal, string(md.Values(mdKey)[0]))

				// Check data. Returns nil if got goodData, error if got badData.
				log.Printf("@@@@ Handler called %+v\n", msg)
				switch v := msg.(*wrapperspb.StringValue).Value; v {
				case goodData:
					goodC <- v
					return nil

				case badData:
					// NOTE: since returnning error will cause message redelivery.
					// Don't block here.
					select {
					case badC <- v:
					default:
					}
					return errors.New("bad data")

				default:
					panic(fmt.Errorf("Unexpect value %s", v))
				}
			},
			SubOptStanAckWait(time.Second), // Short ack wait results in fast redelivery.
		)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Subscribed: %s\n", <-subC)
	}

	// Publish.
	{
		ctx := npmd.NewOutgoingContextWithMD(context.Background(), npmd.NewMetaDataPairs(mdKey, mdVal))

		// Publish good msg data using pb format.
		{
			err = publisher.Publish(ctx, spec, wrapperspb.String(goodData))
			assert.NoError(err)
			log.Printf("Publish and handler got good data: %v.\n", <-goodC)
		}

		// Publish bad msg data using json format and cause redelivery.
		{
			err = publisher.Publish(ctx, spec2, &badData)
			assert.NoError(err)
			log.Printf("Publish and handler got bad data: %v.\n", <-badC)
		}

		// Publish wrong type.
		{
			err = publisher.Publish(ctx, spec, 1)
			assert.Error(err)
		}

		// Publish encode type.
		{
			err = publisher.Publish(ctx, specWrongType, &map[string]interface{}{
				"a": func() {},
			})
			assert.Error(err)
		}

		// Subscribe decode error and cause redelivery.
		{
			err = publisher.Publish(ctx, specWrongType, &map[string]interface{}{
				"a": "b",
			})
			assert.NoError(err)
		}

		// Publish other data to panic and cause redelivery.
		{
			err = publisher.Publish(ctx, spec, wrapperspb.String("panic"))
			assert.NoError(err)
		}
	}

	// First server gone.
	res1.Close()
	log.Printf("Test stan server 1 closed\n")

	// Wait disconnect.
	log.Printf("DurConn disconnected %p.\n", <-disconnectC)

	// Starts the second server using same options.
	var res2 *tststan.Resource
	{
		res2, err = tststan.Run(opts)
		if err != nil {
			log.Panic(err)
		}
		defer res2.Close()
		log.Printf("Test stan server 2 started: %+q\n", res2.NatsURL())
	}

	// Wait re connectioin.
	log.Printf("DurConn reconnected %p.\n", <-connectC)

	// Wait re subscription.
	log.Printf("ReSubscribed: %s\n", <-subC)

	// Now we should see some redelivery.
	for i := 0; i < 3; i++ {
		log.Printf("Bad data redelivery: %v.\n", <-badC)
	}
}

func TestFanout(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestFanout.\n")
	var err error

	// Starts the test server.
	var res *tststan.Resource
	{
		res, err = tststan.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("Test stan server started: %+q\n", res.NatsURL())
	}

	// Common vars.
	spec := MustMsgSpec(
		"test",
		func() interface{} { return wrapperspb.String("") },
	)
	queue := "default"

	{
		// Creates nats connection (with infinite reconnection).
		var nc1 *nats.Conn
		{
			nc1, err = res.NatsClient(
				nats.MaxReconnects(-1),
			)
			if err != nil {
				log.Panic(err)
			}
			defer nc1.Close()
			log.Printf("Connection 1 connected.\n")

			if false {
				// Display raw nats messages flow.
				nc1.Subscribe(">", func(msg *nats.Msg) {
					log.Printf("***** subject=%s reply=%s data=(hex)%x len=%d\n", msg.Subject, msg.Reply, msg.Data, len(msg.Data))
				})
			}
		}

		// Creates the first DurConn.
		var dc1 *DurConn
		subC1 := make(chan MsgSpec, 1)
		{
			dc1, err = NewDurConn(
				nc1,
				res.Options.ClusterId,
				DCOptLogger(newLogger()),
				DCOptSubscribeCb(func(_ stan.Conn, spec MsgSpec) { subC1 <- spec }),
			)
			if err != nil {
				log.Panic(err)
			}
			defer dc1.Close()
			log.Printf("DurConn 1 created.\n")
		}
		publisher1 := PbJsonPublisher(dc1)
		subscriber1 := PbJsonSubscriber(dc1)

		// The first subscribe always returns error cause redelivery.
		sub1Handled := make(chan struct{})
		sub1HandledOnce := sync.Once{}
		{
			err = subscriber1.Subscribe(
				spec,
				queue,
				func(ctx context.Context, msg interface{}) error {
					sub1HandledOnce.Do(func() {
						close(sub1Handled)
					})
					return errors.New("err")
				},
				SubOptStanAckWait(time.Second), // Short ack wait results in fast redelivery.
			)
			if err != nil {
				log.Panic(err)
			}

			log.Printf("Subscribed 1: %s\n", <-subC1)
		}

		// Publish.
		{
			err = publisher1.Publish(context.Background(), spec, wrapperspb.String("123"))
			if err != nil {
				log.Panic(err)
			}
			log.Printf("Published 1\n")
		}

		// Handler 1 handled.
		{
			<-sub1Handled
			log.Printf("Handler 1 handled\n")
		}

		// Close dc1.
		dc1.Close()
	}

	{
		// Creates nats connection (with infinite reconnection).
		var nc2 *nats.Conn
		{
			nc2, err = res.NatsClient(
				nats.MaxReconnects(-1),
			)
			if err != nil {
				log.Panic(err)
			}
			defer nc2.Close()
			log.Printf("Connection 2 connected.\n")

		}
		// Creates the second DurConn.
		var dc2 *DurConn
		subC2 := make(chan MsgSpec, 1)
		{
			dc2, err = NewDurConn(
				nc2,
				res.Options.ClusterId,
				DCOptLogger(newLogger()),
				DCOptSubscribeCb(func(_ stan.Conn, spec MsgSpec) { subC2 <- spec }),
			)
			if err != nil {
				log.Panic(err)
			}
			defer dc2.Close()
			log.Printf("DurConn 2 created.\n")
		}
		subscriber2 := PbJsonSubscriber(dc2)

		// The second subscribe returns nil.
		sub2Handled := make(chan struct{})
		sub2HandledOnce := sync.Once{}
		{
			err = subscriber2.Subscribe(
				spec,
				queue,
				func(ctx context.Context, msg interface{}) error {
					sub2HandledOnce.Do(func() {
						close(sub2Handled)
					})
					return nil
				},
			)
			if err != nil {
				log.Panic(err)
			}

			log.Printf("Subscribed 2: %s\n", <-subC2)
		}

		// Handler 2 handled.
		{
			<-sub2Handled
			log.Printf("Handler 2 handled\n")
		}

	}
}
