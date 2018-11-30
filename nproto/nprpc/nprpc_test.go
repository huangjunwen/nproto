package nprpc

import (
	"context"
	"errors"
	"log"
	"math"
	"testing"
	"time"

	"github.com/huangjunwen/nproto/nproto"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	tstnats "github.com/huangjunwen/tstsvc/nats"
	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
)

func TestNatsRPC(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNatsRPC.\n")
	assert := assert.New(t)
	var err error

	var (
		svcName = "test"
	)

	var (
		notfoundMethod = &nproto.RPCMethod{
			Name: "notfound",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
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
			assert.Equal(svcName, nproto.CurrRPCSvcName(ctx))
			assert.Equal(sqrtMethod, nproto.CurrRPCMethod(ctx))
			input := in.(*wrappers.DoubleValue).Value
			if input < 0 {
				return nil, errors.New("sqrt only accepts non-negative numbers")
			}
			return &wrappers.DoubleValue{Value: math.Sqrt(input)}, nil
		}
	)

	var (
		bgMethod = &nproto.RPCMethod{
			Name: "bg",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &wrappers.StringValue{}
			},
		}
		bgTimeKey    = "bgtime"
		bgCanDoneKey = "bgcandone"
		bgHandler    = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			assert.Equal(svcName, nproto.CurrRPCSvcName(ctx))
			assert.Equal(bgMethod, nproto.CurrRPCMethod(ctx))
			var (
				t       time.Duration
				canDone string
				err     error
			)

			d := nproto.Passthru(ctx)
			{
				v := d[bgTimeKey]
				if v == "" {
					v = "0s"
				}
				t, err = time.ParseDuration(v)
				assert.NoError(err)
			}
			{
				canDone = d[bgCanDoneKey]
				if canDone == "" {
					canDone = "true"
				}
			}

			// Emulate a background work needs to wait t.
			go func() {
				select {
				case <-time.After(t):
					assert.Equal("true", canDone)
					return
				case <-ctx.Done():
					assert.Equal("false", canDone)
					return
				}
			}()

			return &wrappers.StringValue{Value: t.String()}, nil
		}
	)

	// Starts test nats server.
	var res *tstnats.Resource
	{
		res, err = tstnats.Run(nil)
		assert.NoError(err)
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates two connections.
	var nc1, nc2 *nats.Conn
	{
		nc1, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		assert.NoError(err)
		defer nc1.Close()
		log.Printf("Connection 1 connected.\n")

		nc2, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		assert.NoError(err)
		defer nc2.Close()
		log.Printf("Connection 2 connected.\n")
	}

	// Creates rpc server from nc1.
	var server *NatsRPCServer
	{
		server, err = NewNatsRPCServer(nc1)
		assert.NoError(err)
		defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	// RegistSvc.
	assert.NoError(server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
		sqrtMethod: sqrtHandler,
		bgMethod:   bgHandler,
	}))
	defer server.DeregistSvc(svcName)
	log.Printf("Svc registed.\n")

	// Creates two rpc clients.
	var client1, client2 *NatsRPCClient
	{
		client1, err = NewNatsRPCClient(nc1)
		assert.NoError(err)
		defer client1.Close()
		log.Printf("NatsRPCClient 1 created.\n")

		client2, err = NewNatsRPCClient(
			nc2,
			ClientOptJSONEncoding(), // Use JSON encoding.
		)
		assert.NoError(err)
		defer client2.Close()
		log.Printf("NatsRPCClient 2 created.\n")
	}

	// Test rpc calls.
	for _, client := range []*NatsRPCClient{client1, client2} {
		// Test method not found.
		{
			handler := client.MakeHandler(svcName, notfoundMethod)
			{
				output, err := handler(context.Background(), &wrappers.DoubleValue{Value: 0})
				assert.Error(err)
				assert.Nil(output)
			}
		}
		// Test normal return/error return.
		{
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
		// Test context.
		{
			handler := client.MakeHandler(svcName, bgMethod)
			{
				output, err := handler(context.Background(), &empty.Empty{})
				assert.NoError(err)
				assert.Equal("0s", output.(*wrappers.StringValue).Value)
			}
			{
				// Test passthru.
				ctx := nproto.AddPassthru(context.Background(), bgTimeKey, "10ms")
				output, err := handler(ctx, &empty.Empty{})
				assert.NoError(err)
				assert.Equal("10ms", output.(*wrappers.StringValue).Value)
			}
			{
				// Test context timeout.
				ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
				ctx = nproto.AddPassthruDict(ctx, map[string]string{
					bgTimeKey:    "10s", // 10s is much longer than context timeout, thus the bg work can't be done.
					bgCanDoneKey: "false",
				})
				output, err := handler(ctx, &empty.Empty{})
				assert.NoError(err)
				assert.Equal("10s", output.(*wrappers.StringValue).Value)
			}
		}
	}

}

func TestRegist(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestRegist.\n")
	assert := assert.New(t)
	var err error

	// Starts the test nats server.
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

	// Regist should ok.
	assert.NoError(server.RegistSvc("test1", nil))
	assert.Len(server.svcs, 1)

	// Regist should failed since duplication.
	assert.Error(server.RegistSvc("test1", nil))
	assert.Len(server.svcs, 1)

	// Now shut down the connection.
	nc.Close()

	// Regist should failed since connection closed.
	assert.Error(server.RegistSvc("test2", nil))
	assert.Len(server.svcs, 1)
}
