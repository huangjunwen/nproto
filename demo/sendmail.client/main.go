package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/nats-io/nats.go"
	ot "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	"github.com/huangjunwen/nproto/v2/natsrpc"
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
	rpctracing "github.com/huangjunwen/nproto/v2/tracing/rpc"

	"github.com/huangjunwen/nproto/v2/demo/sendmail"
)

const (
	// See docker-compose.yml.
	natsHost               = "localhost"
	natsPort               = 4222
	jaegerServiceName      = "sendmail"
	jaegerReporterEndpoint = "http://localhost:14268/api/traces"
)

func main() {
	var logger logr.Logger
	{
		out := zerolog.NewConsoleWriter()
		out.TimeFormat = time.RFC3339
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

	var nc *nats.Conn
	{
		opts := nats.GetDefaultOptions()
		opts.Url = fmt.Sprintf("nats://%s:%d", natsHost, natsPort)
		opts.MaxReconnect = -1 // Never give up reconnect.
		nc, err = opts.Connect()
		if err != nil {
			panic(err)
		}
		defer nc.Close()
		logger.Info("Connect to nats server ok")
	}

	var tracer ot.Tracer
	{
		var closer io.Closer
		config := &jaegercfg.Configuration{
			ServiceName: jaegerServiceName,
			Sampler: &jaegercfg.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			Reporter: &jaegercfg.ReporterConfig{
				CollectorEndpoint: jaegerReporterEndpoint,
			},
		}
		tracer, closer, err = config.NewTracer()
		if err != nil {
			panic(err)
		}
		defer closer.Close()
		logger.Info("New tracer ok")
	}

	// --- Components ---

	var cc *natsrpc.ClientConn
	{
		cc, err = natsrpc.NewClientConn(nc)
		if err != nil {
			panic(err)
		}
		logger.Info("New natsrpc.ClientConn ok")
	}

	var client nprpc.RPCClient
	{
		client = nprpc.NewRPCClientWithMWs(
			natsrpc.NewPbJsonClient(cc),
			rpctracing.WrapRPCClient(tracer),
		)
		logger.Info("New client ok")
	}

	svc := sendmail.InvokeSendMailSvc(client, sendmail.SvcSpec)

	input := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf(">>> ")
		line, err := input.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				logger.Error(err, "Stdin returns error")
			}
			break
		}

		switch cmd := strings.TrimSpace(line); cmd {
		// help
		default:
			fmt.Printf("  Command list:\n")
			fmt.Printf("    h: Show this help.\n")
			fmt.Printf("    l: List all email entries.\n")
			fmt.Printf("    s: Send a new email.\n")

		// list
		case "l":
			func() {
				span := tracer.StartSpan("List Cmd")
				defer span.Finish()
				ctx := ot.ContextWithSpan(context.Background(), span)

				entries, err := svc.List(ctx, &empty.Empty{})
				if err != nil {
					panic(err)
				}

				for _, entry := range entries.Entries {
					fmt.Printf("  %#v\n", entry)
				}
			}()

		// send
		case "s":

		}
	}

}
