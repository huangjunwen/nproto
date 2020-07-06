package natsrpc

import (
	"bytes"
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
	"github.com/huangjunwen/nproto/v2/enc/jsonenc"
	"github.com/huangjunwen/nproto/v2/enc/pbenc"
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
	w := &bytes.Buffer{}
	if err := pbenc.Default.EncodeData(w, m); err != nil {
		panic(err)
	}
	return &RawData{
		EncoderName: pbenc.Default.Name(),
		Bytes:       w.Bytes(),
	}
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
		sqrtSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "sqrt",
			NewInput: func() interface{} {
				return wrapperspb.Double(0)
			},
			NewOutput: func() interface{} {
				return wrapperspb.Double(0)
			},
		}
		sqrtError   = errors.New("sqrt only accepts non-negative numbers")
		sqrtHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			input := in.(*wrapperspb.DoubleValue).Value
			if input < 0 {
				return nil, sqrtError
			}
			return wrapperspb.Double(math.Sqrt(input)), nil
		}
		sqrtRawDataSpec = NewRawDataRPCSpec(svcName, "sqrt")
	)

	// Test svc not found.
	var (
		svcNotFoundSpec = &RPCSpec{
			SvcName:    svcName + "1",
			MethodName: "svcNotFound",
			NewInput: func() interface{} {
				return wrapperspb.String("")
			},
			NewOutput: func() interface{} {
				return wrapperspb.String("")
			},
		}
	)

	// Test method not found.
	var (
		notFoundSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "notFound",
			NewInput: func() interface{} {
				return wrapperspb.String("")
			},
			NewOutput: func() interface{} {
				return wrapperspb.String("")
			},
		}
	)

	// Test encoder not support.
	var (
		jsonOnlySpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "jsonOnly",
			NewInput: func() interface{} {
				return &emptypb.Empty{}
			},
			NewOutput: func() interface{} {
				return &emptypb.Empty{}
			},
		}
		jsonOnlyHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			return &emptypb.Empty{}, nil
		}
	)

	// Test assert input type/output type.
	var (
		assertTypeSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "assertType",
			NewInput: func() interface{} {
				return wrapperspb.String("")
			},
			NewOutput: func() interface{} {
				return wrapperspb.String("")
			},
		}
		assertTypeHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			// XXX: wrong output type
			return &emptypb.Empty{}, nil
		}
	)

	// Test encoding/decoding.
	var (
		encodeSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "encode",
			NewInput: func() interface{} {
				return &map[string]interface{}{}
			},
			NewOutput: func() interface{} {
				return &map[string]interface{}{}
			},
		}
		encodeHandler = func(ctx context.Context, in interface{}) (interface{}, error) {
			input := in.(*map[string]interface{})
			if (*input)["fn"] != nil {
				// XXX: function can't be encoded
				(*input)["fn"] = func() {}
			}
			return input, nil
		}
		encodeWrongInputSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "encode",
			NewInput: func() interface{} {
				ret := ""
				return &ret
			},
			NewOutput: func() interface{} {
				return &map[string]interface{}{}
			},
		}
		encodeWrongOutputSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "encode",
			NewInput: func() interface{} {
				return &map[string]interface{}{}
			},
			NewOutput: func() interface{} {
				ret := ""
				return &ret
			},
		}
	)

	// Test meta data.
	var (
		metaDataSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "metaData",
			NewInput: func() interface{} {
				return &emptypb.Empty{}
			},
			NewOutput: func() interface{} {
				return wrapperspb.String("")
			},
		}
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
		timeoutSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "timeout",
			NewInput: func() interface{} {
				return &emptypb.Empty{}
			},
			NewOutput: func() interface{} {
				return &emptypb.Empty{}
			},
		}
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
		blockSpec = &RPCSpec{
			SvcName:    svcName,
			MethodName: "block",
			NewInput: func() interface{} {
				return &emptypb.Empty{}
			},
			NewOutput: func() interface{} {
				return &emptypb.Empty{}
			},
		}
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

	// Creates server.
	var (
		server         *Server
		jsonOnlyServer RPCServer
	)

	// Test NewServer with bad options.
	{
		runner, err := limitedrunner.New(
			limitedrunner.MinWorkers(1),
			limitedrunner.MaxWorkers(1),
			limitedrunner.QueueSize(1),
		)
		if err != nil {
			log.Panic(err)
		}

		serverBad, err := NewServer(
			nc1,
			ServerOptRunner(runner),
			ServerOptEncoders(), // <- bad option
		)
		assert.Nil(serverBad)
		assert.Error(err)
	}

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

		server, err = NewServer(
			nc1,
			ServerOptSubjectPrefix(subjectPrefix),
			ServerOptGroup(group),
			ServerOptEncoders(jsonenc.Default, pbenc.Default),
			ServerOptRunner(runner),
			ServerOptLogger(logger),
			ServerOptContext(baseCtx),
		)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()
		log.Printf("natsrpc.Server created.\n")

		jsonOnlyServer = server.MustSubServer(jsonenc.Name)
	}

	// Creates two rpc clients using different encodings.
	var pbClient, jsonClient *Client
	{
		pbClient, err = NewClient(
			nc1,
			ClientOptSubjectPrefix(subjectPrefix),
			ClientOptEncoder(pbenc.Default),
			ClientOptTimeout(time.Second),
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("natsrpc.Client (pb) created.\n")

		jsonClient, err = NewClient(
			nc2,
			ClientOptSubjectPrefix(subjectPrefix),
			ClientOptEncoder(jsonenc.Default),
			ClientOptTimeout(time.Second),
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("natsrpc.Client (json) created.\n")
	}

	// Regist handlers.
	if err := server.RegistHandler(sqrtSpec, sqrtHandler); err != nil {
		log.Panic(err)
	}
	if err := jsonOnlyServer.RegistHandler(jsonOnlySpec, jsonOnlyHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(assertTypeSpec, assertTypeHandler); err != nil {
		log.Panic(err)
	}
	if err := jsonOnlyServer.RegistHandler(encodeSpec, encodeHandler); err != nil {
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

	assert.Error(server.RegistHandler(&RPCSpec{}, nil))

	testCases := []*struct {
		Client   RPCClient
		Spec     *RPCSpec
		GenInput func() (context.Context, interface{})

		Before func(RPCHandler)
		After  func()

		ExpectOutput interface{}
		ExpectError  bool
		AlterCheck   func()
	}{
		// normal case, encoder is pb
		{
			Client: pbClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(9)
			},
			ExpectOutput: wrapperspb.Double(3),
			ExpectError:  false,
		},
		// normal case, encoder is json
		{
			Client: jsonClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(9)
			},
			ExpectOutput: wrapperspb.Double(3),
			ExpectError:  false,
		},
		// handler return error, encoder is pb
		{
			Client: pbClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// handler return error, encoder is json
		{
			Client: jsonClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// use RawData, encoder is pb
		{
			Client: pbClient,
			Spec:   sqrtRawDataSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), pbencRawData(wrapperspb.Double(9))
			},
			ExpectOutput: pbencRawData(wrapperspb.Double(3)),
			ExpectError:  false,
		},
		// use RawData, encoder is json
		{
			Client: jsonClient,
			Spec:   sqrtRawDataSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &RawData{
					EncoderName: jsonenc.Name,
					Bytes:       []byte("9"),
				}
			},
			ExpectOutput: &RawData{
				EncoderName: jsonenc.Name,
				Bytes:       []byte("3"),
			},
			ExpectError: false,
		},
		// svc not found
		{
			Client: pbClient,
			Spec:   svcNotFoundSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("svcNotFound")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// method not found
		{
			Client: pbClient,
			Spec:   notFoundSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("methodNotFound")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call json method with pb client
		{
			Client: pbClient,
			Spec:   jsonOnlySpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &emptypb.Empty{}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call json method with json client
		{
			Client: jsonClient,
			Spec:   jsonOnlySpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &emptypb.Empty{}
			},
			ExpectOutput: &emptypb.Empty{},
			ExpectError:  false,
		},
		// call with wrong input type
		{
			Client: pbClient,
			Spec:   assertTypeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), &emptypb.Empty{}
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// call with wrong output type
		{
			Client: pbClient,
			Spec:   assertTypeSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.String("1")
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// server decode input error
		{
			Client: jsonClient,
			Spec:   encodeWrongInputSpec,
			GenInput: func() (context.Context, interface{}) {
				input := "123"
				return context.Background(), &input
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// server encode output error
		{
			Client: jsonClient,
			Spec:   encodeSpec,
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
			Client: jsonClient,
			Spec:   encodeSpec,
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
			Client: jsonClient,
			Spec:   encodeWrongOutputSpec,
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
			Client: pbClient,
			Spec:   metaDataSpec,
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
			Client: pbClient,
			Spec:   timeoutSpec,
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
			Client: jsonClient,
			Spec:   timeoutSpec,
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
			Client: jsonClient,
			Spec:   timeoutSpec,
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
			Client: jsonClient,
			Spec:   blockSpec,
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

		clientName := "pb"
		if testCase.Client == jsonClient {
			clientName = "json"
		}

		log.Printf("[%d] spec=%+v client=%s\n\tctx=%+v\n\tinput=%T(%+v)\n\toutput=%T(%+v)\n\terr=%+v\n\n",
			i, testCase.Spec, clientName, ctx, input, input, output, output, err)
	}

}
