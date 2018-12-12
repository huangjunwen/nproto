package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

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
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()

	flag.IntVar(&payloadLen, "l", 1000, "Payload length.")
	flag.IntVar(&rpcNum, "n", 1000, "RPC number per client.")
	flag.IntVar(&clientNum, "c", 10, "Client number.")
	flag.IntVar(&timeoutSec, "t", 3, "RPC timeout in seconds.")
	flag.StringVar(&addr, "u", nats.DefaultURL, "gnatsd addr.")
	flag.Parse()

	log.Printf("Payload length (-l): %d\n", payloadLen)
	log.Printf("RPC num per client (-n): %d\n", rpcNum)
	log.Printf("Client number (-c): %d\n", clientNum)
	log.Printf("RPC timeout in seconds (-t): %d\n", timeoutSec)

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

	wg := &sync.WaitGroup{}
	wg.Add(clientNum)
	for i := 0; i < clientNum; i++ {
		go func(i int) {
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

			start := time.Now()
			for j := 0; j < rpcNum; j++ {
				ctx, _ := context.WithTimeout(context.Background(), timeout)
				_, err := svc.Echo(ctx, &benchapi.EchoMsg{
					Msg: payload,
				})
				if err != nil {
					errCnt += 1
				} else {
					succCnt += 1
				}
			}
			dur := time.Since(start)
			avg := dur / time.Duration(rpcNum)

			log.Printf("Client %d: succ=%d err=%d time=%s, avg=%s\n", i, succCnt, errCnt, dur.String(), avg.String())
			wg.Done()
		}(i)
	}

	wg.Wait()
}
