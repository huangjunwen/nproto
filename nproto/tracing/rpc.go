package tracing

import (
	"context"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

type TracedRPCClient struct {
	client nproto.RPCClient
	tracer ot.Tracer
}

type TracedRPCServer struct {
	server nproto.RPCServer
	tracer ot.Tracer
}

var (
	_ nproto.RPCServer = (*TracedRPCServer)(nil)
	_ nproto.RPCClient = (*TracedRPCClient)(nil)
)

func NewTracedRPCClient(client nproto.RPCClient, tracer ot.Tracer) *TracedRPCClient {
	return &TracedRPCClient{
		client: client,
		tracer: tracer,
	}
}

// MakeHandler implements nproto.RPCClient interface.
func (client *TracedRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {

	tracer := client.tracer
	opName := ClientHandlerOpNameFmt(svcName, method)
	handler := client.client.MakeHandler(svcName, method)

	return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
		// Gets current span context as parent. Maybe nil.
		parentSpanCtx := spanCtxFromCtx(ctx)

		// Starts a client span.
		span := tracer.StartSpan(
			opName,
			ot.ChildOf(parentSpanCtx),
			otext.SpanKindRPCClient,
			ComponentTag,
		)
		defer func() {
			setSpanError(span, err)
			span.Finish()
		}()

		// Injects span context.
		md := nproto.MDFromOutgoingContext(ctx).Copy()
		err = injectSpanCtx(tracer, span.Context(), md, TracingMDKey)
		if err != nil {
			return
		}
		ctx = nproto.NewOutgoingContextWithMD(ctx, md)

		// Handles.
		output, err = handler(ctx, input)
		return
	}
}

func NewTracedRPCServer(server nproto.RPCServer, tracer ot.Tracer) *TracedRPCServer {
	return &TracedRPCServer{
		server: server,
		tracer: tracer,
	}
}

// RegistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) RegistSvc(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) error {

	tracer := server.tracer
	methods2 := make(map[*nproto.RPCMethod]nproto.RPCHandler)

	for method, handler := range methods {

		method := method
		handler := handler
		opName := ServerHandlerOpNameFmt(svcName, method)

		methods2[method] = func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Extracts parent span context from client. Maybe nil.
			parentSpanCtx, err := extractSpanCtx(
				tracer,
				nproto.MDFromIncomingContext(ctx),
				TracingMDKey,
			)
			if err != nil {
				return
			}

			// Starts a server span.
			span := tracer.StartSpan(
				opName,
				ot.ChildOf(parentSpanCtx),
				otext.SpanKindRPCServer,
				ComponentTag,
			)
			defer func() {
				setSpanError(span, err)
				span.Finish()
			}()

			// Adds span to context.
			ctx = ot.ContextWithSpan(ctx, span)

			// Handles.
			output, err = handler(ctx, input)
			return

		}

	}

	return server.server.RegistSvc(svcName, methods2)
}

// DeregistSvc implements nproto.RPCServer interface.
func (server *TracedRPCServer) DeregistSvc(svcName string) error {
	return server.server.DeregistSvc(svcName)
}
