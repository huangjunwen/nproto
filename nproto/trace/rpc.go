package trace

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/huangjunwen/nproto/nproto"
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

// MakeHandler implements nproto.RPCClient interface.
func (client *TracedRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {
	fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
	handler := client.Client.MakeHandler(svcName, method)
	return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
		// Extract current span context.
		var parentCtx ot.SpanContext
		if parentSpan := ot.SpanFromContext(ctx); parentSpan != nil {
			parentCtx = parentSpan.Context()
		}

		// Start a client span.
		clientSpan := client.Tracer.StartSpan(
			fullMethodName,
			ot.ChildOf(parentCtx),
			otext.SpanKindRPCClient,
			ComponentTag,
		)
		defer clientSpan.Finish()

		// Inject span context.
		md := nproto.FromOutgoingContext(ctx)
		if md == nil {
			md = nproto.NewMetaDataPairs()
		} else {
			md = md.Copy()
		}
		mdRW := mdReaderWriter{md}
		if err := client.Tracer.Inject(clientSpan.Context(), ot.TextMap, mdRW); err != nil {
			return nil, err
		}

		// Handle.
		output, err = handler(nproto.NewOutgoingContext(ctx, md), input)
		if err != nil {
			otext.Error.Set(clientSpan, true)
			clientSpan.LogFields(otlog.String("event", "error"), otlog.String("message", err.Error()))
		}

		return
	}
}

// RegistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) RegistSvc(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) error {
	methods2 := make(map[*nproto.RPCMethod]nproto.RPCHandler)
	for method, handler := range methods {
		fullMethodName := fmt.Sprintf("%s.%s", svcName, method.Name)
		h := func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Extract span context.
			md := nproto.CurrRPCMetaData(ctx)
			if md == nil {
				md = nproto.NewMetaDataPairs()
			}
			spanCtx, err := server.Tracer.Extract(ot.TextMap, mdReaderWriter{md})
			if err != nil {
				return nil, err
			}

			// Start a server span.
			serverSpan := server.Tracer.StartSpan(
				fullMethodName,
				otext.RPCServerOption(spanCtx),
				ComponentTag,
				otext.SpanKindRPCServer,
			)
			defer serverSpan.Finish()

			// Handle.
			output, err = handler(ot.ContextWithSpan(ctx, serverSpan), input)
			if err != nil {
				otext.Error.Set(serverSpan, true)
				serverSpan.LogFields(otlog.String("event", "error"), otlog.String("message", err.Error()))
			}
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
