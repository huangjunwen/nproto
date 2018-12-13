package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/codahale/hdrhistogram"
	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/nats-io/go-nats"

	benchapi "github.com/huangjunwen/nproto/tests/bench/api"
)

var (
	payloadLen int
	rpcNum     int
	clientNum  int
	timeoutSec int
	addr       string
	cpuprofile string
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()

	flag.IntVar(&payloadLen, "l", 1000, "Payload length.")
	flag.IntVar(&rpcNum, "n", 10000, "Total RPC number.")
	flag.IntVar(&clientNum, "c", 10, "Client number.")
	flag.IntVar(&timeoutSec, "t", 3, "RPC timeout in seconds.")
	flag.StringVar(&addr, "u", nats.DefaultURL, "gnatsd addr.")
	flag.StringVar(&cpuprofile, "cpu", "", "CPU profile file name.")
	flag.Parse()

	log.Printf("Nats URL: %+q\n", addr)
	log.Printf("Payload length (-l): %d\n", payloadLen)
	log.Printf("Total RPC number (-n): %d\n", rpcNum)
	log.Printf("Client number (-c): %d\n", clientNum)
	log.Printf("RPC timeout in seconds (-t): %d\n", timeoutSec)

	var payload string
	{
		p := strings.Builder{}
		for i := 0; i < payloadLen; i++ {
			_, err := p.WriteString("x")
			if err != nil {
				panic(err)
			}
		}
		payload = p.String()
	}
	timeout := time.Duration(timeoutSec) * time.Second
	rpcNumPerClient := rpcNum / clientNum
	durations := make([]time.Duration, clientNum*rpcNumPerClient)

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
	wg.Add(clientNum)

	mu := &sync.Mutex{}
	totalSuccCnt := 0
	totalErrCnt := 0

	start := time.Now()
	for i := 0; i < clientNum; i++ {
		go func(i int) {
			defer func() {
				if err := recover(); err != nil {
					log.Fatal(err)
				}
			}()

			// Filling durations[offset: offset+rpcNumPerClient]
			offset := i * rpcNumPerClient

			nc, err := nats.Connect(
				addr,
				nats.MaxReconnects(-1),
				nats.Name(fmt.Sprintf("client-%d-%d", os.Getpid(), i)),
			)
			if err != nil {
				panic(err)
			}
			defer nc.Close()

			client, err := nprpc.NewNatsRPCClient(nc)
			if err != nil {
				panic(err)
			}
			defer client.Close()

			svc := benchapi.InvokeBench(client, benchapi.SvcName)

			succCnt := 0
			errCnt := 0

			for j := 0; j < rpcNumPerClient; j++ {
				ctx, _ := context.WithTimeout(context.Background(), timeout)

				start := time.Now()
				_, err := svc.Echo(ctx, &benchapi.EchoMsg{
					Msg: payload,
				})
				durations[offset+j] = time.Since(start)

				if err != nil {
					errCnt += 1
				} else {
					succCnt += 1
				}
			}

			mu.Lock()
			totalSuccCnt += succCnt
			totalErrCnt += errCnt
			mu.Unlock()

			wg.Done()
		}(i)
	}

	log.Printf("=== Wating ===\n")
	wg.Wait()
	elapse := time.Since(start)

	if cpuprofile != "" {
		pprof.StopCPUProfile()
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	h := hdrhistogram.New(1, int64(durations[len(durations)-1]), 5)
	for _, d := range durations {
		h.RecordValue(int64(d))
	}

	log.Printf("Elapse=%s Succ=%d Err=%d\n", elapse, totalSuccCnt, totalErrCnt)
	log.Printf("Latency HDR Percentiles:\n")
	log.Printf("10:       %v\n", time.Duration(h.ValueAtQuantile(10)))
	log.Printf("50:       %v\n", time.Duration(h.ValueAtQuantile(50)))
	log.Printf("75:       %v\n", time.Duration(h.ValueAtQuantile(75)))
	log.Printf("90:       %v\n", time.Duration(h.ValueAtQuantile(90)))
	log.Printf("99:       %v\n", time.Duration(h.ValueAtQuantile(99)))
	log.Printf("99.99:    %v\n", time.Duration(h.ValueAtQuantile(99.99)))
	log.Printf("99.999:   %v\n", time.Duration(h.ValueAtQuantile(99.999)))
	log.Printf("99.9999:  %v\n", time.Duration(h.ValueAtQuantile(99.9999)))
	log.Printf("99.99999: %v\n", time.Duration(h.ValueAtQuantile(99.99999)))
	log.Printf("100:      %v\n", time.Duration(h.ValueAtQuantile(100.0)))

}
