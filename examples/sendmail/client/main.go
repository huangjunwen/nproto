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

	"github.com/huangjunwen/nproto/v2/examples/sendmail"
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

		rtt, err := nc.RTT()
		if err != nil {
			panic(err)
		}
		logger.Info("Connect to nats server ok", "rtt", rtt.String())
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
			//Reporter: &jaegercfg.ReporterConfig{
			//	CollectorEndpoint: jaegerReporterEndpoint,
			//},
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

	defer func() {
		e := recover()
		if e == nil {
			return
		}
		err, ok := e.(*readLineError)
		if !ok {
			panic(e)
		}
		if err.Err == io.EOF {
			logger.Info("EOF")
			return
		}
		logger.Error(err.Err, "Read line error")
	}()
	readLine := func() string {
		line, err := input.ReadString('\n')
		if err != nil {
			panic(&readLineError{Err: err})
		}
		return strings.TrimSpace(line)
	}

	for {
		fmt.Printf(">>> ")
		switch cmd := readLine(); cmd {
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

				fmt.Printf("  Found %d entries\n", len(entries.Entries))
				for _, entry := range entries.Entries {
					printEmailEntry(entry)
				}
			}()

		// send
		case "s":
			func() {
				email := &sendmail.Email{}
			AGAIN:
				fmt.Printf(">>>> Input To Addr: ")
				email.ToAddr = readLine()
				fmt.Printf(">>>> Input To Name: ")
				email.ToName = readLine()
				fmt.Printf(">>>> Input Subject: ")
				email.Subject = readLine()
				fmt.Printf(">>>> Input Content Type: ")
				email.ContentType = readLine()
				fmt.Printf(">>>> Input Content: ")
				email.Content = readLine()

				fmt.Printf(">>>> Confirm ? [yN]")
				confirm := readLine()
				if strings.ToLower(confirm) != "y" {
					goto AGAIN
				}

				span := tracer.StartSpan("Send Cmd")
				defer span.Finish()
				ctx := ot.ContextWithSpan(context.Background(), span)

				entry, err := svc.Send(ctx, email)
				if err != nil {
					panic(err)
				}
				printEmailEntry(entry)

			}()

		}
	}

}

func printEmailEntry(entry *sendmail.EmailEntry) {
	fmt.Printf("  Id=%d Status=%s CreatedAt=%q EndedAt=%q FailedReason=%q\n", entry.Id, entry.Status.String(), entry.CreatedAt, entry.EndedAt, entry.FailedReason)
	fmt.Printf("    To=%s<%s>\n", entry.Email.ToName, entry.Email.ToAddr)
	fmt.Printf("    Subject=%s\n", entry.Email.Subject)
	fmt.Printf("    ContentType=%s\n", entry.Email.ContentType)
	fmt.Printf("    Content=%s\n", entry.Email.Content)
	fmt.Printf("\n")
}

type readLineError struct {
	Err error
}

func (err *readLineError) Error() string {
	return err.Err.Error()
}

var (
	_ error = (*readLineError)(nil)
)
