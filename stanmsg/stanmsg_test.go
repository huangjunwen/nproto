package stanmsg

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/huangjunwen/tstsvc"
	tststan "github.com/huangjunwen/tstsvc/stan"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	//"github.com/stretchr/testify/assert"
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
	connectC := make(chan stan.Conn)
	disconnectC := make(chan stan.Conn)
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
	{
		sc := <-connectC
		log.Printf("DurConn connected %p.\n", sc)
	}

	// First server gone.
	res1.Close()
	log.Printf("Test stan server 1 closed\n")

	// Wait disconnect.
	{
		sc := <-disconnectC
		log.Printf("DurConn disconnected %p.\n", sc)
	}

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
	{
		sc := <-connectC
		log.Printf("DurConn reconnect %p.\n", sc)
	}
}
