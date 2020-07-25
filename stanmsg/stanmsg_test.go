package stanmsg

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"
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

func TestDurConnConnect(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestDurConnConnect.\n")
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

func TestDurConnPubSub(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestDurConnPubSub.\n")
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
					panic(errors.New("Impossible branch"))
				}
			},
			SubOptStanAckWait(time.Second), // Short ack wait results in fast redelivery.
		)
		assert.NoError(err)

		log.Printf("Subscribed: %s\n", <-subC)
	}

	// Publish.
	{
		ctx := npmd.NewOutgoingContextWithMD(context.Background(), npmd.NewMetaDataPairs(mdKey, mdVal))

		// Publish good msg data.
		{
			err = publisher.Publish(ctx, spec, wrapperspb.String(goodData))
			assert.NoError(err)
			log.Printf("Publish and handler got good data: %v.\n", <-goodC)
		}

		// Publish bad msg data.
		{
			err = publisher.Publish(ctx, spec, wrapperspb.String(badData))
			assert.NoError(err)
			log.Printf("Publish and handler got bad data: %v.\n", <-badC)
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
