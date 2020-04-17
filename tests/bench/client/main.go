package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"github.com/codahale/hdrhistogram"
	"github.com/huangjunwen/nproto/nproto/natsrpc"
	"github.com/nats-io/nats.go"
	//"go.uber.org/ratelimit"
	"github.com/juju/ratelimit"

	benchapi "github.com/huangjunwen/nproto/tests/bench/api"
)

const fsecs = float64(time.Second)

var (
	addr       string
	payloadLen int
	rpcNum     int
	clientNum  int
	parallel   int
	callRate   int
	timeoutSec int
	cpuprofile string
)

func main() {
	flag.StringVar(&addr, "u", nats.DefaultURL, "gnatsd addr.")
	flag.IntVar(&payloadLen, "l", 1000, "Payload length.")
	flag.IntVar(&rpcNum, "n", 10000, "Total RPC number.")
	flag.IntVar(&clientNum, "c", 10, "Client number.")
	flag.IntVar(&parallel, "p", 10, "Parallel go routines.")
	flag.IntVar(&callRate, "r", 10000, "Target call rate per second.")
	flag.IntVar(&timeoutSec, "t", 3, "RPC timeout in seconds.")
	flag.StringVar(&cpuprofile, "cpu", "", "CPU profile file name.")
	flag.Parse()

	// Prepare.
	var payload []byte
	{
		payload = make([]byte, payloadLen)
		rand.Read(payload)
	}
	rpcNumPerGoroutine := rpcNum / parallel
	rpcNumActual := rpcNumPerGoroutine * parallel
	timeout := time.Duration(timeoutSec) * time.Second
	durations := make([]time.Duration, rpcNumActual)
	//rl := ratelimit.New(callRate)
	rl := ratelimit.NewBucketWithRate(float64(callRate), int64(parallel))
	svcs := make([]benchapi.Bench, clientNum)
	for i := 0; i < clientNum; i++ {
		nc, err := nats.Connect(
			addr,
			nats.MaxReconnects(-1),
			nats.Name(fmt.Sprintf("client-%d-%d", os.Getpid(), i)),
		)
		if err != nil {
			panic(err)
		}
		defer nc.Close()

		client, err := natsrpc.NewNatsRPCClient(nc)
		if err != nil {
			panic(err)
		}

		svcs[i] = benchapi.InvokeBench(client, benchapi.SvcName)
	}

	log.Printf("Nats URL: %+q\n", addr)
	log.Printf("Payload length (-l): %d\n", payloadLen)
	log.Printf("Total RPC number (-n): %d\n", rpcNumActual)
	log.Printf("Client number (-c): %d\n", clientNum)
	log.Printf("Parallel go routines (-p): %d\n", parallel)
	log.Printf("Target call rate per second (-r): %d\n", callRate)
	log.Printf("RPC timeout in seconds (-t): %d\n", timeoutSec)

	// Start.
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(parallel)
	elapseStart := time.Now()
	for i := 0; i < parallel; i++ {
		go func(i int) {
			// Choose one client.
			svc := svcs[i%clientNum]
			// Filling durations[offset: offset+rpcNumPerGoroutine]
			offset := i * rpcNumPerGoroutine

			for j := 0; j < rpcNumPerGoroutine; j++ {
				ctx, _ := context.WithTimeout(context.Background(), timeout)

				//rl.Take()
				rl.Wait(1)
				callStart := time.Now()
				_, err := svc.Echo(ctx, &benchapi.EchoMsg{
					Payload: payload,
				})
				durations[offset+j] = time.Since(callStart)

				if err != nil {
					log.Fatal(err)
				}
			}

			wg.Done()
		}(i)
	}

	// Wait.
	log.Printf("=== Wating ===\n")
	wg.Wait()
	elapse := time.Since(elapseStart)
	if cpuprofile != "" {
		pprof.StopCPUProfile()
	}

	// Post process.
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	// http://vanillajava.blogspot.com/2012/04/what-is-latency-throughput-and-degree.html
	throughput := float64(rpcNumActual) / elapse.Seconds() // How many calls per second.
	m := median(durations)                                 // Median latency value.
	concurrencyActual := throughput * m.Seconds()

	h := hdrhistogram.New(1, int64(durations[len(durations)-1]), 5)
	for _, d := range durations {
		h.RecordValue(int64(d))
	}
	avg := time.Duration(h.Mean())

	log.Printf("Elapse=%v\n", elapse.String())
	log.Printf("Actual call rate=%6.3f RPC/sec\n", throughput)
	log.Printf("Median latency=%v\n", m)
	log.Printf("Avg latency=%v\n", avg)
	log.Printf("Actual concurency=%6.3f\n", concurrencyActual)
	log.Printf("Latency HDR Percentiles:\n")
	log.Printf("10:       %v\n", time.Duration(h.ValueAtQuantile(10)))
	log.Printf("50:       %v\n", time.Duration(h.ValueAtQuantile(50)))
	log.Printf("75:       %v\n", time.Duration(h.ValueAtQuantile(75)))
	log.Printf("80:       %v\n", time.Duration(h.ValueAtQuantile(80)))
	log.Printf("90:       %v\n", time.Duration(h.ValueAtQuantile(90)))
	log.Printf("95:       %v\n", time.Duration(h.ValueAtQuantile(95)))
	log.Printf("99:       %v\n", time.Duration(h.ValueAtQuantile(99)))
	log.Printf("99.99:    %v\n", time.Duration(h.ValueAtQuantile(99.99)))
	log.Printf("99.999:   %v\n", time.Duration(h.ValueAtQuantile(99.999)))
	log.Printf("100:      %v\n", time.Duration(h.ValueAtQuantile(100.0)))

}

func median(durations []time.Duration) time.Duration {
	l := len(durations)
	if l%2 == 0 {
		return (durations[l/2-1] + durations[l/2]) / 2
	}
	return durations[l/2]
}
