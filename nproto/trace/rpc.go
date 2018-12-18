package trace

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	opentracingext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

// TraceRPCClient wraps an RPCClient with an opentracing tracer.
type TraceRPCClient struct {
	Client nproto.RPCClient
	Tracer opentracing.Tracer
}

// TraceRPCServer wraps an RPCServer with an opentracing tracer.
type TraceRPCServer struct {
	Server nproto.RPCServer
	Tracer opentracing.Tracer
}

var (
	_ nproto.RPCServer = (*TraceRPCServer)(nil)
	_ nproto.RPCClient = (*TraceRPCClient)(nil)
)

// MakeHandler implements nproto.RPCClient interface.
func (client *TraceRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {
	fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
	handler := client.Client.MakeHandler(svcName, method)
	return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
		// Extract current span context.
		var parentCtx opentracing.SpanContext
		if parent := opentracing.SpanFromContext(ctx); parent != nil {
			parentCtx = parent.Context()
		}

		// Start a client span.
		child := client.Tracer.StartSpan(
			fullMethodName,
			opentracing.ChildOf(parentCtx),
			opentracingext.SpanKindRPCClient,
			ComponentTag,
		)
		defer child.Finish()

		// Inject span context.
		md := nproto.FromOutgoingContext(ctx)
		if md == nil {
			md = nproto.NewMetaDataPairs()
		} else {
			md = md.Copy()
		}
		mdRW := mdReaderWriter{md}
		if err := client.Tracer.Inject(child.Context(), opentracing.TextMap, mdRW); err != nil {
			return nil, err
		}
		ctx = nproto.NewOutgoingContext(ctx, md)

		return handler(ctx, input)
	}
}

// RegistSvc implements nproto.RPCServer interface.
func (server *TraceRPCServer) RegistSvc(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) error {
	methods2 := make(map[*nproto.RPCMethod]nproto.RPCHandler)
	for method, handler := range methods {
		fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
		h := func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Extract span context.
			md := nproto.CurrRPCMetaData(ctx)
			if md == nil {
				md = nproto.NewMetaDataPairs()
			}
			spanCtx, err := server.Tracer.Extract(opentracing.TextMap, mdReaderWriter{md})
			if err != nil {
				return nil, err
			}

			// Start a server span.
			serverSpan := server.Tracer.StartSpan(
				fullMethodName,
				opentracingext.RPCServerOption(spanCtx),
				ComponentTag,
				opentracingext.SpanKindRPCServer,
			)
			defer serverSpan.Finish()

			ctx = opentracing.ContextWithSpan(ctx, serverSpan)
			return handler(ctx, input)
		}
		methods2[method] = h
	}
	return server.Server.RegistSvc(svcName, methods2)
}

// DeregistSvc implements nproto.RPCServer interface.
func (server *TraceRPCServer) DeregistSvc(svcName string) error {
	return server.Server.DeregistSvc(svcName)
}
