package rpctracing

import (
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/rpc"
	. "github.com/huangjunwen/nproto/v2/tracing"
)

// WrapRPCClient adds opentracing support for RPCClient.
func WrapRPCClient(tracer ot.Tracer) RPCMiddleware {
	return func(spec RPCSpec, handler RPCHandler) RPCHandler {
		opName := ClientOpName(spec)
		return func(ctx context.Context, input interface{}) (output interface{}, err error) {
			parentSpanCtx := SpanCtxFromCtx(ctx)
			if parentSpanCtx == nil {
				return handler(ctx, input)
			}

			span := tracer.StartSpan(
				opName,
				ot.ChildOf(parentSpanCtx),
				otext.SpanKindRPCClient,
				ClientComponentTag,
			)
			defer func() {
				SetSpanError(span, err)
			}()

			md := npmd.MDFromOutgoingContext(ctx)
			md, err = InjectSpanCtx(tracer, span.Context(), md)
			if err != nil {
				return
			}
			ctx = npmd.NewOutgoingContextWithMD(ctx, md)

			return handler(ctx, input)
		}
	}
}

// WrapRPCServer adds opentracing support for RPCServer.
func WrapRPCServer(tracer ot.Tracer) RPCMiddleware {
	return func(spec RPCSpec, handler RPCHandler) RPCHandler {
		opName := ServerOpName(spec)
		return func(ctx context.Context, input interface{}) (output interface{}, err error) {
			parentSpanCtx, err := ExtractSpanCtx(tracer, npmd.MDFromIncomingContext(ctx))
			if err != nil {
				return
			}
			if parentSpanCtx == nil {
				return handler(ctx, input)
			}

			span := tracer.StartSpan(
				opName,
				ot.ChildOf(parentSpanCtx),
				otext.SpanKindRPCServer,
				ServerComponentTag,
			)
			defer func() {
				SetSpanError(span, err)
				span.Finish()
			}()

			return handler(ot.ContextWithSpan(ctx, span), input)
		}
	}

}
