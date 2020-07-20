package natsrpc

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/huangjunwen/golibs/taskrunner/limitedrunner"
	tstnats "github.com/huangjunwen/tstsvc/nats"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/huangjunwen/nproto/v2/enc"
	"github.com/huangjunwen/nproto/v2/enc/pj"
	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

// XXX: assert.Equal(proto.Message, proto.Message) directly may case infinite loop....
func assertEqual(assert *assert.Assertions, v1, v2 interface{}, args ...interface{}) {
	m1, isProtoMsg1 := v1.(proto.Message)
	m2, isProtoMsg2 := v2.(proto.Message)
	if isProtoMsg1 && isProtoMsg2 {
		assert.True(proto.Equal(m1, m2), args...)
		return
	}
	assert.Equal(v1, v2, args...)
}

func pbencRawData(m proto.Message) *RawData {
	ret := &RawData{}
	if err := pj.DefaultPbEncoder.EncodeData(m, &ret.Format, &ret.Bytes); err != nil {
		panic(err)
	}
	return ret
}

func newString(s string) *string {
	return &s
}

func TestRPC(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestRPC.\n")
	var err error
	assert := assert.New(t)

	var (
		svcName                = "test"
		baseCtx, baseCtxCancel = context.WithCancel(context.Background())
	)
	defer baseCtxCancel()

	// Test normal/error.
	var (
		sqrtSpec = MustRPCSpec(
			svcName,
			"sqrt",
			func() interface{} { return wrapperspb.Double(0) },
			func() interface{} { return wrapperspb.Double(0) },
		)
		sqrtError   = errors.New("sqrt only accepts non-negative numbers")
		sqrtHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			input := in.(*wrapperspb.DoubleValue).Value
			if input < 0 {
				return nil, sqrtError
			}
			return wrapperspb.Double(math.Sqrt(input)), nil
		}
		sqrtRawDataSpec = MustRawDataRPCSpec(svcName, "sqrt")
	)

	// Test svc not found.
	var (
		svcNotFoundSpec = MustRPCSpec(
			svcName+"1",
			"svcNotFound",
			func() interface{} { return wrapperspb.String("") },
			func() interface{} { return wrapperspb.String("") },
		)
	)

	// Test method not found.
	var (
		notFoundSpec = MustRPCSpec(
			svcName,
			"notFound",
			func() interface{} { return wrapperspb.String("") },
			func() interface{} { return wrapperspb.String("") },
		)
		notFoundHandler = func(ctx context.Context, input interface{}) (interface{}, error) {
			return input, nil
		}
	)

	// Test assert input type/output type.
	var (
		assertTypeSpec = MustRPCSpec(
			svcName,
			"assertType",
			func() interface{} { return wrapperspb.String("") },
			func() interface{} { return wrapperspb.String("") },
		)
		assertTypeHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			// XXX: wrong output type
			return &emptypb.Empty{}, nil
		}
	)

	// Test encoding/decoding.
	var (
		encodeSpec = MustRPCSpec(
			svcName,
			"encode",
			func() interface{} { return &map[string]interface{}{} },
			func() interface{} { return &map[string]interface{}{} },
		)
		encodeHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			input := in.(*map[string]interface{})
			if (*input)["fn"] != nil {
				// XXX: function can't be encoded
				(*input)["fn"] = func() {}
			}
			return input, nil
		}
		encodeWrongInputSpec = MustRPCSpec(
			svcName,
			"encode",
			func() interface{} { return newString("") },
			func() interface{} { return &map[string]interface{}{} },
		)
		encodeWrongOutputSpec = MustRPCSpec(
			svcName,
			"encode",
			func() interface{} { return &map[string]interface{}{} },
			func() interface{} { return newString("") },
		)
	)

	// Test meta data.
	var (
		metaDataSpec = MustRPCSpec(
			svcName,
			"metaData",
			func() interface{} { return &emptypb.Empty{} },
			func() interface{} { return wrapperspb.String("") },
		)
		metaDataError   = errors.New("No metaData found")
		metaDataHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			md := npmd.MDFromIncomingContext(ctx)
			values := md.Values("metaData")
			if len(values) == 0 {
				return nil, metaDataError
			}
			strs := []string{}
			for _, value := range values {
				strs = append(strs, string(value))
			}

			return wrapperspb.String(strings.Join(strs, " ")), nil
		}
	)

	// Test timeout.
	var (
		timeoutSpec = MustRPCSpec(
			svcName,
			"timeout",
			func() interface{} { return &emptypb.Empty{} },
			func() interface{} { return &emptypb.Empty{} },
		)
		// NOTE: we can't get result from rpc output since client maybe context cancel.
		// So put result here.
		timeoutResultCh = make(chan bool, 1)
		// Returns true if after 1 second, false if context done.
		timeoutHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			select {
			case <-time.After(time.Second):
				timeoutResultCh <- true
			case <-ctx.Done():
				timeoutResultCh <- false
			}
			return &emptypb.Empty{}, nil
		}
	)

	// Test handler block.
	var (
		blockSpec = MustRPCSpec(
			svcName,
			"block",
			func() interface{} { return &emptypb.Empty{} },
			func() interface{} { return &emptypb.Empty{} },
		)
		blockCh      = make(chan struct{})
		blockHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			<-blockCh
			return &emptypb.Empty{}, nil
		}
	)

	// Starts test nats server.
	var res *tstnats.Resource
	{
		res, err = tstnats.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates two connections.
	var nc1, nc2 *nats.Conn
	{
		nc1, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc1.Close()
		log.Printf("Connection 1 connected.\n")

		nc2, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc2.Close()
		log.Printf("Connection 2 connected.\n")
	}

	var (
		subjectPrefix = "natsrpc2"
		group         = "group2"
	)

	// Creates server conn.
	var (
		sc *ServerConn
	)

	{
		runner, err := limitedrunner.New(
			limitedrunner.MinWorkers(1),
			limitedrunner.MaxWorkers(1),
			limitedrunner.QueueSize(1),
		)
		if err != nil {
			log.Panic(err)
		}

		out := zerolog.NewConsoleWriter()
		out.Out = os.Stderr
		lg := zerolog.New(&out).With().Timestamp().Logger()
		logger := (*zerologr.Logger)(&lg)

		sc, err = NewServerConn(
			nc1,
			SCOptSubjectPrefix(subjectPrefix),
			SCOptGroup(group),
			SCOptRunner(runner),
			SCOptLogger(logger),
			SCOptContext(baseCtx),
		)
		if err != nil {
			log.Panic(err)
		}
		defer sc.Close()
		log.Printf("natsrpc.ServerConn created.\n")
	}

	server := sc.Server(pj.DefaultPjDecoder, pj.DefaultPjEncoder)

	// Creates client conns.
	var (
		cc1, cc2 *ClientConn
	)
	{
		cc1, err = NewClientConn(
			nc1,
			CCOptSubjectPrefix(subjectPrefix),
			CCOptTimeout(time.Second),
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("natsrpc.ClientConn 1 created.\n")

		cc2, err = NewClientConn(
			nc2,
			CCOptSubjectPrefix(subjectPrefix),
			CCOptTimeout(time.Second),
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("natsrpc.ClientConn 2 created.\n")
	}

	pbClient := cc1.Client(pj.DefaultPbEncoder, pj.DefaultPjDecoder)

	jsonClient := cc2.Client(pj.DefaultJsonEncoder, pj.DefaultPjDecoder)

	// Regist handlers.
	if err := server.RegistHandler(sqrtSpec, sqrtHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(assertTypeSpec, assertTypeHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(encodeSpec, encodeHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(metaDataSpec, metaDataHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(timeoutSpec, timeoutHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(blockSpec, blockHandler); err != nil {
		log.Panic(err)
	}

	{
		// Regist and deregist
		if err := server.RegistHandler(notFoundSpec, notFoundHandler); err != nil {
			log.Panic(err)
		}
		if err := server.RegistHandler(notFoundSpec, nil); err != nil {
			log.Panic(err)
		}
	}

	testCases := []*struct {
		Client     RPCClient
		ClientName string
		Spec       RPCSpec
		GenInput   func() (context.Context, interface{})

		Before func(RPCHandler)
		After  func()

		AlterCheck   func()
		ExpectOutput interface{}
		ExpectError  bool
	}{
		// normal case, encoder is pb
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(9)
			},
			ExpectOutput: wrapperspb.Double(3),
			ExpectError:  false,
		},
		// normal case, encoder is json
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(9)
			},
			ExpectOutput: wrapperspb.Double(3),
			ExpectError:  false,
		},
		// handler return error, encoder is pb
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// handler return error, encoder is json
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// use RawData, encoder is pb
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       sqrtRawDataSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), pbencRawData(wrapperspb.Double(9))
			},
			ExpectOutput: pbencRawData(wrapperspb.Double(3)),
			ExpectError:  false,
		},
		// use RawData, encoder is json
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       sqrtRawDataSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &RawData{
					Format: JsonFormat,
					Bytes:  []byte("9"),
				}
			},
			ExpectOutput: &RawData{
				Format: JsonFormat,
				Bytes:  []byte("3"),
			},
			ExpectError: false,
		},
		// svc not found
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       svcNotFoundSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("svcNotFound")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// method not found
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       notFoundSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("methodNotFound")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call with wrong input type
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       assertTypeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &emptypb.Empty{}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call with wrong output type
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       assertTypeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("1")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// server decode input error
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       encodeWrongInputSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), newString("123")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// server encode output error
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       encodeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &map[string]interface{}{
					"fn": "1",
				}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// client encode input error
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       encodeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &map[string]interface{}{
					"fn": func() {},
				}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// client decode output error
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       encodeWrongOutputSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &map[string]interface{}{
					"a": "b",
				}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call with meta data.
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       metaDataSpec,
			GenInput: func() (context.Context, interface{}) {
				ctx := npmd.NewOutgoingContextWithMD(
					context.Background(),
					npmd.MetaData{"metaData": [][]byte{[]byte("1"), []byte("2")}},
				)
				return ctx, &emptypb.Empty{}
			},
			ExpectOutput: wrapperspb.String("1 2"),
			ExpectError:  false,
		},
		// call with timeout 2 seconds.
		{
			Client:     pbClient,
			ClientName: "pb",
			Spec:       timeoutSpec,
			GenInput: func() (context.Context, interface{}) {
				ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
				return ctx, &emptypb.Empty{}
			},
			AlterCheck: func() {
				assert.True(<-timeoutResultCh)
			},
		},
		// call with timeout 0.2 seconds.
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       timeoutSpec,
			GenInput: func() (context.Context, interface{}) {
				ctx, _ := context.WithTimeout(context.Background(), 200*time.Millisecond)
				return ctx, &emptypb.Empty{}
			},
			AlterCheck: func() {
				assert.False(<-timeoutResultCh)
			},
		},
		// call with already timeout.
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       timeoutSpec,
			GenInput: func() (context.Context, interface{}) {
				ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
				time.Sleep(100 * time.Millisecond)
				return ctx, &emptypb.Empty{}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// block handlers.
		{
			Client:     jsonClient,
			ClientName: "json",
			Spec:       blockSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &emptypb.Empty{}
			},
			Before: func(handler RPCHandler) {
				// One for worker and one for queue. Then then next request will be unable to handle.
				go handler(context.Background(), &emptypb.Empty{})
				time.Sleep(500 * time.Millisecond)
				go handler(context.Background(), &emptypb.Empty{})
				time.Sleep(500 * time.Millisecond)
			},
			After: func() {
				blockCh <- struct{}{}
				blockCh <- struct{}{}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
	}
	for i, testCase := range testCases {
		testCase := testCase
		handler := testCase.Client.MakeHandler(testCase.Spec)

		if testCase.Before != nil {
			testCase.Before(handler)
		}
		ctx, input := testCase.GenInput()
		output, err := handler(ctx, input)
		if testCase.After != nil {
			testCase.After()
		}

		if testCase.AlterCheck != nil {
			testCase.AlterCheck()
		} else {
			assertEqual(assert, testCase.ExpectOutput, output, "test case %d", i)
			if testCase.ExpectError {
				assert.Error(err, "test case %d", i)
			} else {
				assert.NoError(err, "test case %d", i)
			}
		}

		log.Printf("[%d] spec=%+v client=%s\n\tctx=%+v\n\tinput=%T(%+v)\n\toutput=%T(%+v)\n\terr=%+v\n\n",
			i, testCase.Spec, testCase.ClientName, ctx, input, input, output, output, err)
	}

}
