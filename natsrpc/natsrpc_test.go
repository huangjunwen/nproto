package natsrpc

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"testing"
	"time"

	tstnats "github.com/huangjunwen/tstsvc/nats"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

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

func TestRPC(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestRPC.\n")
	var err error
	assert := assert.New(t)

	var (
		svcName = "test"
	)

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
	)

	// Test svc not found.
	var (
		notFound2Spec = &RPCSpec{
			SvcName:    svcName + "1",
			MethodName: "notFound",
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

	// Test encoder.
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

	{
		server, err = NewServer(
			nc1,
			ServerOptSubjectPrefix(subjectPrefix),
			ServerOptGroup(group),
			ServerOptEncoders(jsonenc.Default, pbenc.Default),
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
	if err := server.RegistHandler(metaDataSpec, metaDataHandler); err != nil {
		log.Panic(err)
	}
	if err := server.RegistHandler(timeoutSpec, timeoutHandler); err != nil {
		log.Panic(err)
	}

	testCases := []*struct {
		Client   RPCClient
		Spec     *RPCSpec
		GenInput func() (context.Context, interface{})

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
		// error case, encoder is pb
		{
			Client: pbClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// error case, encoder is json
		{
			Client: jsonClient,
			Spec:   sqrtSpec,
			GenInput: func() (context.Context, interface{}) {
				return context.Background(), wrapperspb.Double(-9)
			},
			ExpectOutput: nil,
			ExpectError:  true,
		},
		// svcName not found
		{
			Client: pbClient,
			Spec:   notFound2Spec,
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
	}
	for i, testCase := range testCases {
		testCase := testCase
		handler := testCase.Client.MakeHandler(testCase.Spec)
		ctx, input := testCase.GenInput()

		output, err := handler(ctx, input)

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
		log.Printf("Test case %d:\n\tctx=%+v\n\tinput=%T(%+v)\n\toutput=%T(%+v)\n\terr=%+v\n\n",
			i, ctx, input, input, output, output, err)
	}
}