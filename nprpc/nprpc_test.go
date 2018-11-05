package nprpc

import (
	"context"
	"errors"
	"log"
	"math"
	"testing"

	"github.com/huangjunwen/nproto"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	tstnats "github.com/huangjunwen/tstsvc/nats"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
)

var (
	svcName = "test"
)

var (
	sqrtMethod = &nproto.RPCMethod{
		Name: "sqrt",
		NewInput: func() proto.Message {
			return &wrappers.DoubleValue{}
		},
		NewOutput: func() proto.Message {
			return &wrappers.DoubleValue{}
		},
	}
	sqrtHandler = func(ctx context.Context, in proto.Message) (proto.Message, error) {
		// Return error if input is negative number.
		input := in.(*wrappers.DoubleValue).Value
		if input < 0 {
			return nil, errNegative
		}

		// Return.
		output := math.Sqrt(input)
		return &wrappers.DoubleValue{Value: output}, nil
	}
	errNegative = errors.New("sqrt only accepts non-negative numbers")
)

func TestNatsRPC(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNatsRPC.\n")
	assert := assert.New(t)
	var err error

	// Starts test nats server.
	var res *tstnats.Resource
	{
		res, err = tstnats.Run(nil)
		assert.NoError(err)
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates connection.
	var nc *nats.Conn
	{
		nc, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		assert.NoError(err)
		defer nc.Close()
		log.Printf("Connection connected.\n")
	}

	// Creates rpc server.
	var server *NatsRPCServer
	{
		server, err = NewNatsRPCServer(nc)
		assert.NoError(err)
		defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	// RegistSvc.
	assert.NoError(server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
		sqrtMethod: sqrtHandler,
	}))
	defer server.DeregistSvc(svcName)
	log.Printf("Svc registed.\n")

	// Use the same connection to create a rpc client then test.
	{
		client, err := NewNatsRPCClient(nc)
		assert.NoError(err)
		defer client.Close()
		log.Printf("NatsRPCClient 1 created.\n")

		handler := client.MakeHandler(svcName, sqrtMethod)

		{
			output, err := handler(context.Background(), &wrappers.DoubleValue{Value: 81})
			assert.NoError(err)
			assert.Equal(float64(9), output.(*wrappers.DoubleValue).Value)
		}
		{
			output, err := handler(context.Background(), &wrappers.DoubleValue{Value: -1})
			assert.Error(err)
			assert.Nil(output)
		}
	}

	// Creates another connection to create a rpc client then test.
	{
		nc2, err := res.NatsClient(
			nats.MaxReconnects(-1),
		)
		assert.NoError(err)
		defer nc2.Close()
		log.Printf("Another connection connected.\n")

		client, err := NewNatsRPCClient(
			nc2,
			ClientOptJSONEncoding(), // Use JSON encoding.
		)
		assert.NoError(err)
		defer client.Close()
		log.Printf("NatsRPCClient 2 created.\n")

		handler := client.MakeHandler(svcName, sqrtMethod)

		{
			output, err := handler(context.Background(), &wrappers.DoubleValue{Value: 81})
			assert.NoError(err)
			assert.Equal(float64(9), output.(*wrappers.DoubleValue).Value)
		}
		{
			output, err := handler(context.Background(), &wrappers.DoubleValue{Value: -1})
			assert.Error(err)
			assert.Nil(output)
		}
	}
}
