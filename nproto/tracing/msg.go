package tracing

import (
	"context"

	_ "github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

// TracedMsgPublisher adds opentracing support for MsgPublisher.
func TracedMsgPublisher(tracer ot.Tracer, downstream bool) nproto.MsgPublisherMiddleware {
	return func(next nproto.MsgPublisherFunc) nproto.MsgPublisherFunc {
		return func(ctx context.Context, subject string, msgData []byte) (err error) {
			var (
				parentSpanCtx ot.SpanContext
				spanRef       ot.SpanReference
			)

			// Gets parent span reference.
			if downstream {
				parentSpanCtx, err = extractSpanCtx(
					tracer,
					nproto.MDFromOutgoingContext(ctx),
				)
				if err != nil {
					return
				}
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx)
				}
			} else {
				parentSpanCtx = spanCtxFromCtx(ctx)
				if parentSpanCtx != nil {
					spanRef = ot.ChildOf(parentSpanCtx)
				}
			}

			// Do not start new span if no parent span.
			if parentSpanCtx == nil {
				err = next(ctx, subject, msgData)
				return
			}

			// Starts a producer span.
			span := tracer.StartSpan(
				PublishOpNameFmt(subject),
				spanRef,
				otext.SpanKindProducer,
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

			// Publish.
			err = next(ctx, subject, msgData)
			return
		}
	}
}

// TracedMsgAsyncPublisher adds opentracing support for MsgAsyncPublisher.
func TracedMsgAsyncPublisher(tracer ot.Tracer, downstream bool) nproto.MsgAsyncPublisherMiddleware {
	return func(next nproto.MsgAsyncPublisherFunc) nproto.MsgAsyncPublisherFunc {
		return func(ctx context.Context, subject string, msgData []byte, cb func(error)) (err error) {
			var (
				parentSpanCtx ot.SpanContext
				spanRef       ot.SpanReference
			)

			// Gets parent span reference.
			if downstream {
				parentSpanCtx, err = extractSpanCtx(
					tracer,
					nproto.MDFromOutgoingContext(ctx),
				)
				if err != nil {
					return
				}
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx)
				}
			} else {
				parentSpanCtx = spanCtxFromCtx(ctx)
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx) // Use FollowFrom since PublishAsync is async op.
				}
			}

			// Do not start new span if no parent span.
			if parentSpanCtx == nil {
				err = next(ctx, subject, msgData, cb)
				return
			}

			// Starts a producer span.
			span := tracer.StartSpan(
				PublishAsyncOpNameFmt(subject),
				spanRef,
				otext.SpanKindProducer,
				ComponentTag,
			)
			fin := func(err error) {
				setSpanError(span, err)
				span.Finish()
			}

			// Injects span context.
			md := nproto.MDFromOutgoingContext(ctx)
			md, err = injectSpanCtx(tracer, span.Context(), md)
			if err != nil {
				fin(err)
				return
			}
			ctx = nproto.NewOutgoingContextWithMD(ctx, md)

			// PublishAsync.
			if err = next(ctx, subject, msgData, func(cbErr error) {
				cb(cbErr)
				fin(cbErr)
			}); err != nil {
				fin(err)
			}
			return
		}
	}
}

func TracedMsgSubscriber(tracer ot.Tracer) nproto.MsgMiddleware {
	return func(subject, queue string, handler nproto.MsgHandler) nproto.MsgHandler {
		opName := SubscriberHandlerOpNameFmt(subject, queue)
		return func(ctx context.Context, msgData []byte) (err error) {
			// Gets parent span context.
			parentSpanCtx, err := extractSpanCtx(
				tracer,
				nproto.MDFromIncomingContext(ctx),
			)
			if err != nil {
				return
			}

			// Do not start new span if no parent span.
			if parentSpanCtx == nil {
				err = handler(ctx, msgData)
				return
			}

			// Starts a consumer span.
			span := tracer.StartSpan(
				opName,
				ot.FollowsFrom(parentSpanCtx),
				otext.SpanKindConsumer,
				ComponentTag,
			)
			defer func() {
				setSpanError(span, err)
				span.Finish()
			}()

			// Adds span to context.
			ctx = ot.ContextWithSpan(ctx, span)

			// Handles.
			err = handler(ctx, msgData)
			return
		}
	}
}
