package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/huangjunwen/nproto/nproto/nprpc"
	"github.com/huangjunwen/nproto/nproto/trace"
	"github.com/nats-io/go-nats"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	traceapi "github.com/huangjunwen/nproto/tests/trace/api"
)

var (
	svc traceapi.Trace // client service
)

type Trace struct{}

func (t Trace) Recursive(ctx context.Context, input *traceapi.RecursiveRequest) (output *traceapi.RecursiveReply, err error) {
	if input.Depth < 0 {
		return nil, fmt.Errorf("Expect postive depth, but got %d", input.Depth)
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

	var nc *nats.Conn
	{
		nc, err = nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NATS connected.\n")
		defer nc.Close()

	}

	var tracer opentracing.Tracer
	{
		var tracerCloser io.Closer
		tracer, tracerCloser, err = NewTracer(traceapi.SvcName)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("Tracer created.\n")
		defer tracerCloser.Close()
	}

	{
		server, err := nprpc.NewNatsRPCServer(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCServer created.\n")
		defer server.Close()

		tserver := trace.NewTracedRPCServer(server, tracer)
		if err := traceapi.ServeTrace(tserver, traceapi.SvcName, Trace{}); err != nil {
			log.Panic(err)
		}
	}

	{
		client, err := nprpc.NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCClient created.\n")
		defer client.Close()

		tclient := trace.NewTracedRPCClient(client, tracer)
		svc = traceapi.InvokeTrace(tclient, traceapi.SvcName)
	}

	{
		for _, runCase := range []struct {
			Depth        int32
			ExpectResult int32
			ExpectError  bool
		}{
			{0, 1, false},
			{3, 4, false},
			{-1, 0, true},
		} {
			reply, err := svc.Recursive(context.Background(), &traceapi.RecursiveRequest{
				Depth: runCase.Depth,
			})
			if runCase.ExpectError {
				if err == nil {
					log.Panic("Expect error for run case %v, but got nil", runCase)
				}
			} else {
				if reply.Result != runCase.ExpectResult {
					log.Panic("Expect result %v, but got %v", runCase.ExpectError, reply.Result)
				}
			}

		}
	}

}
