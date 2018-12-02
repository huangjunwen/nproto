package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	mathapi "github.com/huangjunwen/nproto/tests/math/api"

	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/nats-io/go-nats"
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
	for {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		args := make([]float64, rand.Intn(3))
		for i := 0; i < len(args); i++ {
			args[i] = rand.Float64()
		}
		log.Printf("Calling Sum(%v)\n", args)

		sum, err := svc.Sum(ctx, &mathapi.SumRequest{Args: args})
		if err != nil {
			log.Printf("Return error: %s\n", err)
		} else {
			log.Printf("Return sum: %v\n", sum.Sum)
		}

		time.Sleep(time.Second)
	}
}
