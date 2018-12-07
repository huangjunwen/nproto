package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/nats-io/go-nats"

	mathapi "github.com/huangjunwen/nproto/tests/math/api"
)

const (
	seqKey = "seq"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("NATS connected.\n")
	defer nc.Close()

	client, err := nprpc.NewNatsRPCClient(nc)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("NatsRPCClient created.\n")
	defer client.Close()

	svc := mathapi.InvokeMath(client, mathapi.SvcName)
	seq := 1
	for {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		ctx = nproto.NewOutgoingContext(ctx, nproto.NewMetaDataPairs(seqKey, fmt.Sprintf("%d", seq)))
		args := make([]float64, rand.Intn(3))
		for i := 0; i < len(args); i++ {
			args[i] = float64(rand.Intn(100))
		}
		log.Printf("Calling Sum(%v) seq: %+q\n", args, nproto.FromOutgoingContext(ctx).Get(seqKey))

		sum, err := svc.Sum(ctx, &mathapi.SumRequest{Args: args})
		if err != nil {
			log.Printf("Return error: %s\n", err)
		} else {
			log.Printf("Return sum: %v\n", sum.Sum)
		}

		time.Sleep(time.Second)
		seq += 1
	}
}
