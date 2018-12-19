package trace

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

var (
	rpcComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nproto.rpc",
	}
)

// TracedRPCClient wraps an RPCClient with an opentracing tracer.
type TracedRPCClient struct {
	Client nproto.RPCClient
	Tracer ot.Tracer
}

// TracedRPCServer wraps an RPCServer with an opentracing tracer.
type TracedRPCServer struct {
	Server nproto.RPCServer
	Tracer ot.Tracer
}

var (
	_ nproto.RPCServer = (*TracedRPCServer)(nil)
	_ nproto.RPCClient = (*TracedRPCClient)(nil)
)

// NewTracedRPCClient creates a new TracedRPCClient.
func NewTracedRPCClient(client nproto.RPCClient, tracer ot.Tracer) *TracedRPCClient {
	return &TracedRPCClient{
		Client: client,
		Tracer: tracer,
	}
}

// MakeHandler implements nproto.RPCClient interface.
func (client *TracedRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {
	fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
	handler := client.Client.MakeHandler(svcName, method)
	return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
		// Extract current span context, maybe nil.
		curSpanCtx := spanCtxFromCtx(ctx)

		// Start a client span.
		clientSpan := client.Tracer.StartSpan(
			fullMethodName,
			ot.ChildOf(curSpanCtx),
			otext.SpanKindRPCClient,
			rpcComponentTag,
		)
		defer clientSpan.Finish()

		// Inject span context.
		md, err := injectSpanCtx(
			client.Tracer,
			clientSpan.Context(),
			nproto.FromOutgoingContext(ctx),
		)
		if err != nil {
			return nil, err
		}
		ctx = nproto.NewOutgoingContext(ctx, md)

		// Handle.
		output, err = handler(ctx, input)
		setSpanError(clientSpan, err)
		return
	}
}

// NewTracedRPCServer creates a new TracedRPCServer.
func NewTracedRPCServer(server nproto.RPCServer, tracer ot.Tracer) *TracedRPCServer {
	return &TracedRPCServer{
		Server: server,
		Tracer: tracer,
	}
}

// RegistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) RegistSvc(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) error {
	methods2 := make(map[*nproto.RPCMethod]nproto.RPCHandler)
	for method, handler := range methods {
		fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
		h := func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Extract current span context, maybe nil.
			curSpanCtx, err := extractSpanCtx(
				server.Tracer,
				nproto.CurrRPCMetaData(ctx),
			)
			if err != nil {
				return nil, err
			}

			// Start a server span.
			serverSpan := server.Tracer.StartSpan(
				fullMethodName,
				ot.ChildOf(curSpanCtx),
				rpcComponentTag,
				otext.SpanKindRPCServer,
			)
			defer serverSpan.Finish()
			ctx = ot.ContextWithSpan(ctx, serverSpan)

			// Handle.
			output, err = handler(ctx, input)
			setSpanError(serverSpan, err)
			return

		}
		methods2[method] = h
	}
	return server.Server.RegistSvc(svcName, methods2)
}

// DeregistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) DeregistSvc(svcName string) error {
	return server.Server.DeregistSvc(svcName)
}
