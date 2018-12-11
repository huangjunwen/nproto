package bench

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/nats-io/go-nats"

	benchapi "github.com/huangjunwen/nproto/tests/bench/api"
)

const (
	SvcName = "bench"
)

type Bench struct{}

func (svc Bench) Nop(ctx context.Context, input *empty.Empty) (output *empty.Empty, err error) {
	return input, nil
}

func (svc Bench) Echo(ctx context.Context, input *benchapi.EchoMsg) (output *benchapi.EchoMsg, err error) {
	return input, nil
}

var (
	_ benchapi.Bench = Bench{}
)

var (
	serverNum = 100
	clientNum = 100
)

func BenchmarkBench(b *testing.B) {
	// Server side.
	for i := 0; i < serverNum; i++ {
		// Creates connection.
		nc, err := nats.Connect(
			nats.DefaultURL,
			nats.MaxReconnects(-1),
			nats.Name(fmt.Sprintf("server-%d", i)),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()

		// Creates server.
		server, err := nprpc.NewNatsRPCServer(nc)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()

		// Regist service.
		if err := benchapi.ServeBench(server, SvcName, Bench{}); err != nil {
			log.Panic(err)
		}
	}

	// Client side.
	var svcs []benchapi.Bench
	for i := 0; i < clientNum; i++ {
		// Creates connection.
		nc, err := nats.Connect(
			nats.DefaultURL,
			nats.MaxReconnects(-1),
			nats.Name(fmt.Sprintf("client-%d", i)),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()

		// Creates the client.
		client, err := nprpc.NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		defer client.Close()

		// Creates client service.
		svcs = append(svcs, benchapi.InvokeBench(client, SvcName))
	}
	chooseSvc := func() benchapi.Bench {
		return svcs[rand.Intn(len(svcs))]
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.Run("Nop", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			svc := chooseSvc()
			for pb.Next() {
				_, err := svc.Nop(context.Background(), &empty.Empty{})
				if err != nil {
					log.Panic(err)
				}
			}
		})
	})

	b.Run("Echo", func(b *testing.B) {

		subBench := func(n int) {
			b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
				var x strings.Builder
				for i := 0; i < n; i++ {
					_, err := x.WriteString("x")
					if err != nil {
						log.Panic(err)
					}
				}
				s := x.String()

				b.RunParallel(func(pb *testing.PB) {
					svc := chooseSvc()
					for pb.Next() {
						_, err := svc.Echo(context.Background(), &benchapi.EchoMsg{
							Msg: s,
						})
						if err != nil {
							log.Panic(err)
						}

					}
				})
			})
		}

		subBench(1)
		subBench(10)
		subBench(100)
		subBench(1000)
		subBench(5000)
		subBench(10000)

	})
}
