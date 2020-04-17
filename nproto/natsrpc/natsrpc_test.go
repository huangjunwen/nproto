package natsrpc

import (
	"context"
	"errors"
	"log"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	tstnats "github.com/huangjunwen/tstsvc/nats"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/helpers/taskrunner"
	"github.com/huangjunwen/nproto/nproto"
)

func TestNatsRPC(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNatsRPC.\n")
	assert := assert.New(t)
	var err error

	var (
		svcName = "test"
	)

	// This method is used to test method not found.
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

	// This method is used to test normal case.
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
			input := in.(*wrappers.DoubleValue).Value
			if input < 0 {
				return nil, errors.New("sqrt only accepts non-negative numbers")
			}
			return &wrappers.DoubleValue{Value: math.Sqrt(input)}, nil
		}
	)

	// This method is used to test timeout and meta data.
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
		bgWg         = &sync.WaitGroup{} // Used to wait background job.
		bgHandler    = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			var (
				t       time.Duration
				canDone string
				err     error
			)

			md := nproto.MDFromIncomingContext(ctx)
			if md == nil {
				md = nproto.EmptyMD
			}
			// Get time to wait.
			{
				v := "0s"
				vs := md.Values(bgTimeKey)
				if len(vs) != 0 {
					v = string(vs[0])
				}

				t, err = time.ParseDuration(v)
				assert.NoError(err)
			}

			// Get expect result.
			{
				canDone = "true"
				vs := md.Values(bgCanDoneKey)
				if len(vs) != 0 {
					canDone = string(vs[0])
				}
			}

			// Start a background job to wait for some time or context timeout.
			bgWg.Add(1)
			go func() {
				select {
				case <-time.After(t):
					// If wait time is shorter.
					assert.Equal("true", canDone)
				case <-ctx.Done():
					// If dead line is shorter.
					assert.Equal("false", canDone)
				}
				bgWg.Done()
			}()

			return &wrappers.StringValue{Value: t.String()}, nil
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

	var subjectPrefix = "xxx"
	var group = "yyy"

	// Creates rpc server from nc1.
	var server *NatsRPCServer
	{
		server, err = NewNatsRPCServer(
			nc1,
			ServerOptSubjectPrefix(subjectPrefix),
			ServerOptLogger(nil),
			ServerOptGroup(group),
		)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	// RegistSvc.
	{
		if err = server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
			sqrtMethod: sqrtHandler,
			bgMethod:   bgHandler,
		}); err != nil {
			log.Panic(err)
		}
		defer server.DeregistSvc(svcName)
		log.Printf("Svc registered.\n")
	}

	// Creates two rpc clients using different encodings.
	var client1, client2 *NatsRPCClient
	{
		client1, err = NewNatsRPCClient(
			nc1,
			ClientOptPBEncoding(),
			ClientOptSubjectPrefix(subjectPrefix),
		)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCClient 1 created.\n")

		client2, err = NewNatsRPCClient(
			nc2,
			ClientOptJSONEncoding(),
			ClientOptSubjectPrefix(subjectPrefix),
		)
		if err != nil {
			log.Panic(err)
		}
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
		// Test normal method.
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
		// Test metadata.
		{
			handler := client.MakeHandler(svcName, bgMethod)
			{
				// Test without metadata.
				output, err := handler(context.Background(), &empty.Empty{})
				assert.NoError(err)
				assert.Equal("0s", output.(*wrappers.StringValue).Value)
				bgWg.Wait()
			}
			{
				// Test metadata: wait time shorter than context deadline (since context is context.Background()).
				ctx := nproto.NewOutgoingContextWithMD(context.Background(), nproto.NewMetaDataPairs(bgTimeKey, "10ms"))
				output, err := handler(ctx, &empty.Empty{})
				assert.NoError(err)
				assert.Equal("10ms", output.(*wrappers.StringValue).Value)
				bgWg.Wait()
			}
			{
				// Test context timeout: wait time longer than context deadline.
				ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
				ctx = nproto.NewOutgoingContextWithMD(ctx, nproto.NewMetaDataPairs(
					bgTimeKey, "10s",
					bgCanDoneKey, "false",
				))
				output, err := handler(ctx, &empty.Empty{})
				assert.NoError(err)
				assert.Equal("10s", output.(*wrappers.StringValue).Value)
				bgWg.Wait()
			}
		}
	}

}

func TestSvcUnavailable(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestSvcUnavailable.\n")
	assert := assert.New(t)
	var err error

	// Starts the test nats server.
	var res *tstnats.Resource
	{
		res, err = tstnats.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates connection.
	var nc *nats.Conn
	{
		nc, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Connection connected.\n")
	}

	// Creates rpc server.
	var server *NatsRPCServer
	// Create a runner that can only handle one job a time.
	var runner = taskrunner.NewLimitedRunner(1, 0)
	{
		server, err = NewNatsRPCServer(
			nc,
			ServerOptTaskRunner(runner),
		)
		if err != nil {
			log.Panic(err)
		}
		// NOTE: don't call Close() here since the later code will make a never-finish call.
		// defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	// Creates rpc client.
	var client *NatsRPCClient
	{
		client, err = NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCClient created.\n")
	}

	var (
		svcName = "busy"
	)

	var (
		busyMethod = &nproto.RPCMethod{
			Name: "Busy",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
		busyc       = make(chan struct{}, 10)
		busyHandler = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			busyc <- struct{}{}
			time.Sleep(10000 * time.Second) // Never finish.
			return &empty.Empty{}, nil
		}
	)

	// RegistSvc.
	{
		if err = server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
			busyMethod: busyHandler,
		}); err != nil {
			log.Panic(err)
		}
		defer server.DeregistSvc(svcName)
		log.Printf("Svc registered.\n")
	}

	// MakeHandler.
	handler := client.MakeHandler(svcName, busyMethod)

	// Make the first call. It should never finish.
	go handler(context.Background(), &empty.Empty{})
	<-busyc

	output, err := handler(context.Background(), &empty.Empty{})
	assert.Equal(ErrSvcUnavailable.Error(), err.Error())
	assert.Nil(output)

}

func TestSvcContext(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestSvcContext.\n")
	assert := assert.New(t)
	var err error

	// Starts the test nats server.
	var res *tstnats.Resource
	{
		res, err = tstnats.Run(nil)
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates connection.
	var nc *nats.Conn
	{
		nc, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Connection connected.\n")
	}

	// Creates rpc server.
	var server *NatsRPCServer

	// Create a cancelable context.
	ctx, cancel := context.WithCancel(context.Background())
	{
		server, err = NewNatsRPCServer(
			nc,
			ServerOptContext(ctx),
		)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	// Creates rpc client.
	var client *NatsRPCClient
	{
		client, err = NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("NatsRPCClient created.\n")
	}

	var (
		svcName = "longlive"
		errMsg  = "After context done"
	)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	var (
		longliveMethod = &nproto.RPCMethod{
			Name: "Longlive",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
		longliveHandler = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			go func() {
				t := time.Now()
				server.Close()
				log.Printf("server.Close elapsed: %s\n", time.Since(t))
				wg.Done()
			}()
			<-ctx.Done()
			return nil, errors.New(errMsg)
		}
	)

	// RegistSvc.
	{
		if err = server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
			longliveMethod: longliveHandler,
		}); err != nil {
			log.Panic(err)
		}
		defer server.DeregistSvc(svcName)
		log.Printf("Svc registered.\n")
	}

	// MakeHandler.
	handler := client.MakeHandler(svcName, longliveMethod)

	go func() {
		_, err = handler(context.Background(), &empty.Empty{})
		wg.Done()
	}()

	time.Sleep(2 * time.Second)
	cancel()

	wg.Wait()
	assert.Equal(errMsg, err.Error())
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
		if err != nil {
			log.Panic(err)
		}
		defer res.Close()
		log.Printf("Test nats server started: %+q\n", res.NatsURL())
	}

	// Creates connection.
	var nc *nats.Conn
	{
		nc, err = res.NatsClient(
			nats.MaxReconnects(-1),
		)
		if err != nil {
			log.Panic(err)
		}
		defer nc.Close()
		log.Printf("Connection connected.\n")
	}

	// Creates rpc server.
	var server *NatsRPCServer
	{
		server, err = NewNatsRPCServer(nc)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()
		log.Printf("NatsRPCServer created.\n")
	}

	var (
		dupMethod = &nproto.RPCMethod{
			Name: "dup",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
		dupMethod2 = &nproto.RPCMethod{
			Name: "dup",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
		dupHandler = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			return &empty.Empty{}, nil
		}
	)

	{
		// Regist should ok.
		assert.NoError(server.RegistSvc("test1", nil))
		assert.Len(server.svcs, 1)

		// Regist should failed since service name duplication.
		assert.Error(server.RegistSvc("test1", nil))
		assert.Len(server.svcs, 1)
	}

	{
		// Regist should failed since method name duplication.
		assert.Error(server.RegistSvc("test2", map[*nproto.RPCMethod]nproto.RPCHandler{
			dupMethod:  dupHandler,
			dupMethod2: dupHandler,
		}))
		assert.Len(server.svcs, 1)
	}

	{
		// Now shut down the connection.
		nc.Close()

		// Regist should failed since connection closed.
		assert.Error(server.RegistSvc("test3", nil))
		assert.Len(server.svcs, 1)
	}

	{
		// Now close the server.
		assert.NoError(server.Close())
		assert.Len(server.svcs, 0)

		// Regist should failed since server closed.
		assert.Error(server.RegistSvc("test4", nil))
		assert.Len(server.svcs, 0)
	}

}

func BenchmarkNatsRPC(b *testing.B) {
	var err error

	var (
		svcName = "ping"
	)

	var (
		pingMethod = &nproto.RPCMethod{
			Name: "ping",
			NewInput: func() proto.Message {
				return &empty.Empty{}
			},
			NewOutput: func() proto.Message {
				return &empty.Empty{}
			},
		}
		pingHandler = func(ctx context.Context, in proto.Message) (proto.Message, error) {
			return &empty.Empty{}, nil
		}
	)

	nc, err := nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
	if err != nil {
		log.Panic(err)
	}
	defer nc.Close()

	var server *NatsRPCServer
	{
		server, err = NewNatsRPCServer(nc)
		if err != nil {
			log.Panic(err)
		}
		defer server.Close()
	}

	{
		if err = server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
			pingMethod: pingHandler,
		}); err != nil {
			log.Panic(err)
		}
		defer server.DeregistSvc(svcName)
	}

	var client *NatsRPCClient
	{
		client, err = NewNatsRPCClient(nc)
		if err != nil {
			log.Panic(err)
		}
	}

	ping := client.MakeHandler(svcName, pingMethod)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ping(context.Background(), &empty.Empty{})
			if err != nil {
				log.Panic(err)
			}
		}
	})
}
