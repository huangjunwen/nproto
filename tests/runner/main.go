package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/huangjunwen/nproto/helpers/taskrunner"
	"github.com/huangjunwen/nproto/nproto/nprpc"
	nats "github.com/nats-io/nats.go"

	runnerapi "github.com/huangjunwen/nproto/tests/runner/api"
)

type Runner struct{}

func (r Runner) Sleep(ctx context.Context, input *duration.Duration) (output *empty.Empty, err error) {
	dur, err := ptypes.Duration(input)
	if err != nil {
		panic(err)
	}

	time.Sleep(dur)
	log.Printf("Server: Sleep %s\n", dur.String())

	return &empty.Empty{}, nil
}

func main() {
	// Creates the connection.
	nc, err := nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
	if err != nil {
		log.Panic(err)
	}
	defer nc.Close()
	log.Printf("NATS connected.\n")

	// Creates runner.
	runner := taskrunner.NewLimitedRunner(2, 2)
	defer runner.Close()

	// Creates the server.
	server, err := nprpc.NewNatsRPCServer(
		nc,
		nprpc.ServerOptTaskRunner(runner),
	)
	if err != nil {
		log.Panic(err)
	}
	defer server.Close()
	log.Printf("NatsRPCServer created.\n")

	// Regist service.
	if err := runnerapi.ServeRunner(server, runnerapi.SvcName, Runner{}); err != nil {
		log.Panic(err)
	}
	log.Printf("Svc registered.\n")

	// Creates the client.
	client, err := nprpc.NewNatsRPCClient(nc)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("NatsRPCClient created.\n")

	// Creates client service.
	svc := runnerapi.InvokeRunner(client, runnerapi.SvcName)

	// Calls.
	n := 10
	wg := &sync.WaitGroup{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			dur := ptypes.DurationProto(time.Duration(i) * time.Second)
			_, err := svc.Sleep(context.Background(), dur)
			if err != nil {
				log.Printf("Client: Call %d return error %s\n", i, err)
			}
			wg.Done()
		}(i + 1)
	}

	// Sleep a while
	time.Sleep(100 * time.Millisecond)

	// Close and wait.
	log.Printf(">> Start to close\n")
	runner.Close()
	log.Printf(">> Closed\n")

	wg.Wait()
}
