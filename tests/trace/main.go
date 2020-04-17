package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/natsrpc"
	"github.com/huangjunwen/nproto/nproto/stanmsg"
	"github.com/huangjunwen/nproto/nproto/tracing"
	nats "github.com/nats-io/nats.go"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	traceapi "github.com/huangjunwen/nproto/tests/trace/api"
)

var (
	publisher *nproto.PBPublisher
	svc       traceapi.Trace // client service
	wait      chan struct{}
)

type Trace struct{}

func (t Trace) Recursive(ctx context.Context, input *traceapi.RecursiveRequest) (output *traceapi.RecursiveReply, err error) {
	if input.Depth < 0 {
		err := fmt.Errorf("Expect 0 or positive depth, but got %d", input.Depth)
		publisher.Publish(ctx, traceapi.SubjName, &traceapi.RecursiveDepthNegative{
			Depth: input.Depth,
		})
		log.Println(err)
		return nil, err
	}

	result := int32(0)
	if input.Depth > 0 {
		subReply, err := svc.Recursive(ctx, &traceapi.RecursiveRequest{
			Depth: input.Depth - 1,
		})
		if err != nil {
			return nil, err
		}
		result = subReply.Result
	}
	return &traceapi.RecursiveReply{
		Result: result + 1,
	}, nil
}

func HandleRecursiveNegDepth(ctx context.Context, msg *traceapi.RecursiveDepthNegative) error {
	log.Printf("Got negative recursive depth: %d\n", msg.Depth)
	time.Sleep(2 * time.Second)
	close(wait)
	return nil
}

func NewTracer(service string) (opentracing.Tracer, io.Closer, error) {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
	}

	return cfg.New(service, jaegercfg.Logger(jaeger.StdLogger))
}

func main() {
	var err error

	wait = make(chan struct{})

	var nc *nats.Conn
	{
		nc, err = nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NATS connected.\n")
		defer func() {
			nc.Close()
			log.Printf("NATS closed.\n")
		}()
	}

	var tracer opentracing.Tracer
	{
		var tracerCloser io.Closer
		tracer, tracerCloser, err = NewTracer(traceapi.SvcName)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("Tracer created.\n")
		defer func() {
			tracerCloser.Close()
			log.Printf("Tracer closed.\n")
		}()
	}

	{
		server, err := natsrpc.NewNatsRPCServer(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCServer created.\n")
		defer func() {
			server.Close()
			log.Printf("NatsRPCServer closed.\n")
		}()

		tserver := nproto.NewRPCServerWithMWs(server, tracing.WrapRPCServer(tracer))
		if err := traceapi.ServeTrace(tserver, traceapi.SvcName, Trace{}); err != nil {
			log.Panic(err)
		}
	}

	{
		client, err := natsrpc.NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCClient created.\n")

		tclient := nproto.NewRPCClientWithMWs(client, tracing.WrapRPCClient(tracer))
		svc = traceapi.InvokeTrace(tclient, traceapi.SvcName)
	}

	var dc *stanmsg.DurConn
	{
		dc, err = stanmsg.NewDurConn(nc, "test-cluster")
		if err != nil {
			log.Panic(err)
		}
		log.Printf("DurConn created.\n")
		defer func() {
			dc.Close()
			log.Printf("DurConn closed.\n")
		}()
		defer dc.Close()
	}

	{
		publisher = &nproto.PBPublisher{
			Publisher: nproto.NewMsgPublisherWithMWs(dc, tracing.WrapMsgPublisher(tracer, false)),
		}
		log.Printf("Publisher created.\n")
	}

	{
		subscriber := nproto.NewMsgSubscriberWithMWs(dc, tracing.WrapMsgSubscriber(tracer))
		log.Printf("Subscriber created.\n")
		err = traceapi.SubscribeRecursiveDepthNegative(
			subscriber,
			traceapi.SubjName,
			"default",
			HandleRecursiveNegDepth,
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("Subscribed.\n")
	}

	// Wait a while.
	time.Sleep(time.Second)

	{
		for _, runCase := range []struct {
			Depth        int32
			ExpectResult int32
			ExpectError  bool
		}{
			{0, 1, false},
			{3, 4, false},
			{-3, 0, true},
		} {
			func() {
				ctx := context.Background()
				span := tracer.StartSpan("nproto tests trace")
				defer span.Finish()
				ctx = opentracing.ContextWithSpan(ctx, span)

				reply, err := svc.Recursive(ctx, &traceapi.RecursiveRequest{
					Depth: runCase.Depth,
				})
				if runCase.ExpectError {
					if err == nil {
						log.Panicf("Expect error for run case %v, but got nil", runCase)
					}
				} else {
					if reply.Result != runCase.ExpectResult {
						log.Panicf("Expect result %v, but got %v", runCase.ExpectError, reply.Result)
					}
				}
			}()

		}
	}

	<-wait

	log.Printf("End..\n")

}
