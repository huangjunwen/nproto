package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/huangjunwen/nproto/nproto/taskrunner"
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

func main() {
	var (
		payloadLen        int
		rpcNum            int
		clientConcurrency int
		serverConcurrency int
		clientConnNum     int
		serverConnNum     int
		timeoutSec        int
		cpuProfile        string
		memProfile        string
	)
	flag.IntVar(&payloadLen, "l", 1000, "Payload length.")
	flag.IntVar(&rpcNum, "n", 1000, "RPC number per client.")
	flag.IntVar(&clientConcurrency, "c", 10, "Client concurrency (client go routines).")
	flag.IntVar(&serverConcurrency, "s", 1000, "Server concurrency (maximum go routines for server handlers).")
	flag.IntVar(&clientConnNum, "cn", 10, "Client nats connection number.")
	flag.IntVar(&serverConnNum, "sn", 10, "Server nats connection number.")
	flag.IntVar(&timeoutSec, "t", 3, "RPC timeout in seconds.")
	flag.StringVar(&cpuProfile, "cpu", "", "CPU profile.")
	flag.StringVar(&memProfile, "mem", "", "Memory profile.")
	flag.Parse()

	log.Printf("Payload length (-l): %d\n", payloadLen)
	log.Printf("RPC num (per client) (-n): %d\n", rpcNum)
	log.Printf("Client concurrency (-c): %d\n", clientConcurrency)
	log.Printf("Server concurrency (-s): %d\n", serverConcurrency)
	log.Printf("Client connection num (-cn): %d\n", clientConnNum)
	log.Printf("Server connection num (-sn): %d\n", serverConnNum)
	log.Printf("RPC timeout (in seconds) (-t): %d\n", timeoutSec)
	log.Printf(
		"Total bytes will be: %d(Client concurrency) x %d(RPC num) x %d(Payload length) = %d\n",
		clientConcurrency,
		rpcNum,
		payloadLen,
		clientConcurrency*rpcNum*payloadLen)
	log.Printf("Preparing ...\n")

	// Setup server side.
	runner := taskrunner.NewLimitedRunner(serverConcurrency, -1)
	defer runner.Close()

	for i := 0; i < serverConnNum; i++ {
		nc, err := nats.Connect(
			nats.DefaultURL,
			nats.MaxReconnects(-1),
			nats.Name(fmt.Sprintf("server-%d", i)),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()

		server, err := nprpc.NewNatsRPCServer(
			nc,
			nprpc.ServerOptTaskRunner(runner),
		)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()

		if err := benchapi.ServeBench(server, SvcName, Bench{}); err != nil {
			log.Panic(err)
		}
	}

	// Setup client side.
	var svcs []benchapi.Bench
	for i := 0; i < clientConnNum; i++ {
		nc, err := nats.Connect(
			nats.DefaultURL,
			nats.MaxReconnects(-1),
			nats.Name(fmt.Sprintf("client-%d", i)),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()

		client, err := nprpc.NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		defer client.Close()

		svcs = append(svcs, benchapi.InvokeBench(client, SvcName))
	}
	chooseSvc := func() benchapi.Bench {
		return svcs[rand.Intn(len(svcs))]
	}
	var payload string
	{
		p := strings.Builder{}
		for i := 0; i < payloadLen; i++ {
			_, err := p.WriteString("x")
			if err != nil {
				log.Panic(err)
			}
		}
		payload = p.String()
	}
	timeout := time.Duration(timeoutSec) * time.Second

	// Run test.
	wg := &sync.WaitGroup{}
	wg.Add(clientConcurrency)

	mu := &sync.Mutex{}
	succCntSum := 0
	errCntSum := 0
	avgSum := time.Duration(0)

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Panic(err)
		}
		defer f.Close()

		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	start := time.Now()
	for i := 0; i < clientConcurrency; i++ {
		go func() {
			succCnt := 0
			errCnt := 0
			svc := chooseSvc()

			t := time.Now()
			for j := 0; j < rpcNum; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				_, err := svc.Echo(ctx, &benchapi.EchoMsg{
					Msg: payload,
				})
				if err != nil {
					errCnt += 1
				} else {
					succCnt += 1
				}
				cancel()
			}
			dur := time.Since(t)
			avg := dur / time.Duration(rpcNum)

			mu.Lock()
			succCntSum += succCnt
			errCntSum += errCnt
			avgSum += avg
			mu.Unlock()

			wg.Done()
		}()
	}

	log.Printf("Waiting ...\n")
	wg.Wait()
	totalDur := time.Since(start)
	avg := avgSum / time.Duration(clientConcurrency)

	log.Printf("Total time: %s\n", totalDur.String())
	log.Printf("Avg time: %s\n", avg.String())
	log.Printf("Total RPC num: %dx%d=%d\n", rpcNum, clientConcurrency, rpcNum*clientConcurrency)
	log.Printf("Total success RPC num: %d\n", succCntSum)
	log.Printf("Total error RPC num: %d\n", errCntSum)

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			log.Panic(err)
		}
		defer f.Close()

		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Panic(err)
		}
	}
}
