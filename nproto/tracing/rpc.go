package tracing

import (
	"context"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

// TracedRPCClient adds opentracing support for RPCClient.
func TracedRPCClient(tracer ot.Tracer) nproto.RPCMiddleware {
	return func(svcName string, method *nproto.RPCMethod, handler nproto.RPCHandler) nproto.RPCHandler {
		opName := ClientHandlerOpNameFmt(svcName, method)
		return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Gets current span context as parent. Maybe nil.
			parentSpanCtx := spanCtxFromCtx(ctx)

			// Do not start new span if no parent span.
			if parentSpanCtx == nil {
				output, err = handler(ctx, input)
				return
			}

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
			md := nproto.MDFromOutgoingContext(ctx)
			md, err = injectSpanCtx(tracer, span.Context(), md)
			if err != nil {
				return
			}
			ctx = nproto.NewOutgoingContextWithMD(ctx, md)

			// Handles.
			output, err = handler(ctx, input)
			return
		}
	}
}

// TracedRPCServer adds opentracing support for RPCServer.
func TracedRPCServer(tracer ot.Tracer) nproto.RPCMiddleware {
	return func(svcName string, method *nproto.RPCMethod, handler nproto.RPCHandler) nproto.RPCHandler {
		opName := ServerHandlerOpNameFmt(svcName, method)
		return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Extracts parent span context from client. Maybe nil.
			parentSpanCtx, err := extractSpanCtx(tracer, nproto.MDFromIncomingContext(ctx))
			if err != nil {
				return
			}

			// Do not start new span if no parent span.
			if parentSpanCtx == nil {
				output, err = handler(ctx, input)
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

}
