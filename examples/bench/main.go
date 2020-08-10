package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"github.com/codahale/hdrhistogram"
	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/juju/ratelimit"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/huangjunwen/nproto/v2/natsrpc"
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
)

const (
	// See docker-compose.yml.
	natsHost = "localhost"
	natsPort = 4222
)

var (
	payloadLen int
	rpcNum     int
	clientNum  int
	serverNum  int
	parallel   int
	callRate   int
	timeoutSec int
	cpuProfile string
)

func main() {
	flag.IntVar(&payloadLen, "l", 1000, "Payload length.")
	flag.IntVar(&clientNum, "c", 10, "Client number.")
	flag.IntVar(&serverNum, "s", 10, "Client number.")
	flag.IntVar(&rpcNum, "n", 10000, "Total RPC number.")
	flag.IntVar(&parallel, "p", 10, "Number of go routines to invoke RPCs.")
	flag.IntVar(&callRate, "r", 100000, "Call rate limit per second.")
	flag.IntVar(&timeoutSec, "t", 3, "RPC timeout in seconds.")
	flag.StringVar(&cpuProfile, "cpu", "", "CPU profile file name.")
	flag.Parse()

	var logger logr.Logger
	{
		out := zerolog.NewConsoleWriter()
		out.TimeFormat = time.RFC3339Nano
		out.Out = os.Stderr
		lg := zerolog.New(&out).With().Timestamp().Logger()
		logger = (*zerologr.Logger)(&lg)
	}

	var err error
	defer func() {
		if e := recover(); e != nil {
			err, ok := e.(error)
			if !ok {
				err = fmt.Errorf("%+v", e)
			}
			logger.Error(err, "panic", true)
		}
	}()

	// Prepare rpc params.
	spec := nprpc.MustRPCSpec(
		"bench",
		"echo",
		func() interface{} {
			return wrapperspb.Bytes(nil)
		},
		func() interface{} {
			return wrapperspb.Bytes(nil)
		},
	)
	handler := func(ctx context.Context, input interface{}) (interface{}, error) {
		return input, nil
	}
	var input *wrapperspb.BytesValue
	{
		data := make([]byte, payloadLen)
		if _, err = rand.Read(data); err != nil {
			panic(err)
		}
		input = wrapperspb.Bytes(data)
	}

	// Prepare bench params.
	rpcNumPerGoroutine := rpcNum / parallel
	rpcNumActual := rpcNumPerGoroutine * parallel
	timeout := time.Duration(timeoutSec) * time.Second
	durations := make([]time.Duration, rpcNumActual)
	rl := ratelimit.NewBucketWithRate(float64(callRate), int64(parallel))
	logger.Info("Payload Len(-l)     :", "value", payloadLen)
	logger.Info("RPC Num(-n)         :", "value", rpcNumActual)
	logger.Info("Parallel(-p)        :", "value", parallel)
	logger.Info("Call Rate Limit(-r) :", "value", callRate)
	logger.Info("Timeout(-t)         :", "value", timeout.String())

	// Start serverNum servers.
	for i := 0; i < serverNum; i++ {

		// nats connection.
		var nc *nats.Conn
		{
			opts := nats.GetDefaultOptions()
			opts.Url = fmt.Sprintf("nats://%s:%d", natsHost, natsPort)
			opts.MaxReconnect = -1
			nc, err = opts.Connect()
			if err != nil {
				panic(err)
			}
			defer nc.Close()
		}

		var sc *natsrpc.ServerConn
		{
			sc, err = natsrpc.NewServerConn(
				nc,
				natsrpc.SCOptLogger(logger),
			)
			if err != nil {
				panic(err)
			}
			defer sc.Close()
		}

		server := natsrpc.NewPbJsonServer(sc)
		if err = server.RegistHandler(spec, handler); err != nil {
			panic(err)
		}
	}
	logger.Info("Servers ready", "serverNum", serverNum)

	// Make clientNum clients.
	handlers := make([]nprpc.RPCHandler, 0, clientNum)
	for i := 0; i < clientNum; i++ {

		// nats connection.
		var nc *nats.Conn
		{
			opts := nats.GetDefaultOptions()
			opts.Url = fmt.Sprintf("nats://%s:%d", natsHost, natsPort)
			opts.MaxReconnect = -1
			nc, err = opts.Connect()
			if err != nil {
				panic(err)
			}
			defer nc.Close()
		}

		var cc *natsrpc.ClientConn
		{
			cc, err = natsrpc.NewClientConn(
				nc,
				natsrpc.CCOptTimeout(2*time.Second),
			)
			if err != nil {
				panic(err)
			}
		}

		client := natsrpc.NewPbJsonClient(cc)
		handler := client.MakeHandler(spec)
		handlers = append(handlers, handler)
	}
	logger.Info("Clients ready", "clientNum", clientNum)

	// Prof.
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
	}

	// Start.
	wg := &sync.WaitGroup{}
	wg.Add(parallel)
	elapseStart := time.Now()
	for i := 0; i < parallel; i++ {
		go func(i int) {
			defer wg.Done()

			// Choose one client.
			handler := handlers[i%clientNum]
			// Filling durations[offset: offset+rpcNumPerGoroutine]
			offset := i * rpcNumPerGoroutine

			for j := 0; j < rpcNumPerGoroutine; j++ {
				func(j int) {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					rl.Wait(1)
					callStart := time.Now()
					_, err := handler(ctx, input)
					if err != nil {
						panic(err)
					}
					durations[offset+j] = time.Since(callStart)
				}(j)
			}

		}(i)
	}

	logger.Info("===Waiting===")
	wg.Wait()

	// Post calculation.
	elapse := time.Since(elapseStart)
	if cpuProfile != "" {
		pprof.StopCPUProfile()
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	h := hdrhistogram.New(int64(durations[0]), int64(durations[len(durations)-1]), 5)
	for _, d := range durations {
		h.RecordValue(int64(d))
	}
	avg := time.Duration(h.Mean())

	// http://vanillajava.blogspot.com/2012/04/what-is-latency-throughput-and-degree.html
	throughput := float64(rpcNumActual) / elapse.Seconds() // How many calls per second.
	concurrency := throughput * avg.Seconds()

	logger.Info("Elapse                                :", "value", elapse.String())
	logger.Info("Avg Latency                           :", "value", avg.String())
	logger.Info("Throughput  (=RPC Num/Elapse)         :", "value", throughput)
	logger.Info("Concurrency (=Throughput*Avg Latency) :", "value", concurrency)
	logger.Info("10                                    :", "value", time.Duration(h.ValueAtQuantile(10)).String())
	logger.Info("50                                    :", "value", time.Duration(h.ValueAtQuantile(50)).String())
	logger.Info("75                                    :", "value", time.Duration(h.ValueAtQuantile(75)).String())
	logger.Info("80                                    :", "value", time.Duration(h.ValueAtQuantile(80)).String())
	logger.Info("85                                    :", "value", time.Duration(h.ValueAtQuantile(85)).String())
	logger.Info("90                                    :", "value", time.Duration(h.ValueAtQuantile(90)).String())
	logger.Info("95                                    :", "value", time.Duration(h.ValueAtQuantile(95)).String())
	logger.Info("99                                    :", "value", time.Duration(h.ValueAtQuantile(99)).String())
	logger.Info("99.9                                  :", "value", time.Duration(h.ValueAtQuantile(99.9)).String())
	logger.Info("99.99                                 :", "value", time.Duration(h.ValueAtQuantile(99.99)).String())
	logger.Info("99.999                                :", "value", time.Duration(h.ValueAtQuantile(99.999)).String())
	logger.Info("100                                   :", "value", time.Duration(h.ValueAtQuantile(100)).String())

}

func median(durations []time.Duration) time.Duration {
	l := len(durations)
	if l%2 == 0 {
		return (durations[l/2-1] + durations[l/2]) / 2
	}
	return durations[l/2]
}
