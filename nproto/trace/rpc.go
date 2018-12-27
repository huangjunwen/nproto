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
	client nproto.RPCClient
	tracer ot.Tracer
}

// TracedRPCServer wraps an RPCServer with an opentracing tracer.
type TracedRPCServer struct {
	server nproto.RPCServer
	tracer ot.Tracer
}

var (
	_ nproto.RPCServer = (*TracedRPCServer)(nil)
	_ nproto.RPCClient = (*TracedRPCClient)(nil)
)

// NewTracedRPCClient creates a new TracedRPCClient.
func NewTracedRPCClient(client nproto.RPCClient, tracer ot.Tracer) *TracedRPCClient {
	return &TracedRPCClient{
		client: client,
		tracer: tracer,
	}
}

// MakeHandler implements nproto.RPCClient interface.
func (client *TracedRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {
	fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
	handler := client.client.MakeHandler(svcName, method)
	return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
		// Extract current span context, maybe nil.
		curSpanCtx := spanCtxFromCtx(ctx)

		// Start a client span.
		clientSpan := client.tracer.StartSpan(
			fullMethodName,
			ot.ChildOf(curSpanCtx),
			otext.SpanKindRPCClient,
			rpcComponentTag,
		)
		defer clientSpan.Finish()

		// Inject span context.
		md, err := injectSpanCtx(
			client.tracer,
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
		server: server,
		tracer: tracer,
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
				server.tracer,
				nproto.CurrRPCMetaData(ctx),
			)
			if err != nil {
				return nil, err
			}

			// Start a server span.
			serverSpan := server.tracer.StartSpan(
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
	return server.server.RegistSvc(svcName, methods2)
}

// DeregistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) DeregistSvc(svcName string) error {
	return server.server.DeregistSvc(svcName)
}
