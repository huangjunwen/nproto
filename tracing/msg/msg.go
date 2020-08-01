package msgtracing

import (
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
	. "github.com/huangjunwen/nproto/v2/tracing"
)

// WrapMsgPublisher adds opentracing support for MsgPublisher.
// Pass true to `downstream` if the publisher is used as msg pipe downstream.
func WrapMsgPublisher(tracer ot.Tracer, downstream bool) MsgPublisherMiddleware {
	return func(next MsgPublisherFunc) MsgPublisherFunc {
		return func(ctx context.Context, spec MsgSpec, msg interface{}) (err error) {
			var (
				parentSpanCtx ot.SpanContext
				spanRef       ot.SpanReference
			)

			if downstream {
				// Start span following upstream.
				parentSpanCtx, err = ExtractSpanCtx(
					tracer,
					npmd.MDFromOutgoingContext(ctx),
				)
				if err != nil {
					return
				}
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx)
				}
			} else {
				// Start span as child of current span.
				parentSpanCtx = SpanCtxFromCtx(ctx)
				if parentSpanCtx != nil {
					spanRef = ot.ChildOf(parentSpanCtx)
				}
			}

			if parentSpanCtx == nil {
				err = next(ctx, spec, msg)
				return
			}

			span := tracer.StartSpan(
				PublisherOpName(spec),
				spanRef,
				otext.SpanKindProducer,
				PublisherComponentTag,
			)
			defer func() {
				SetSpanError(span, err)
				span.Finish()
			}()

			md := npmd.MDFromOutgoingContext(ctx)
			md, err = InjectSpanCtx(tracer, span.Context(), md)
			if err != nil {
				return
			}
			ctx = npmd.NewOutgoingContextWithMD(ctx, md)

			err = next(ctx, spec, msg)
			return
		}
	}
}

// WrapMsgAsyncPublisher adds opentracing support for MsgAsyncPublisher.
// Pass true to `downstream` if the publisher is used as msg pipe downstream.
func WrapMsgAsyncPublisher(tracer ot.Tracer, downstream bool) MsgAsyncPublisherMiddleware {
	return func(next MsgAsyncPublisherFunc) MsgAsyncPublisherFunc {
		return func(ctx context.Context, spec MsgSpec, msg interface{}, cb func(error)) (err error) {
			var (
				parentSpanCtx ot.SpanContext
				spanRef       ot.SpanReference
			)

			if downstream {
				// Start span following upstream.
				parentSpanCtx, err = ExtractSpanCtx(
					tracer,
					npmd.MDFromOutgoingContext(ctx),
				)
				if err != nil {
					return
				}
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx)
				}
			} else {
				// Start span following current span since this is an async op.
				parentSpanCtx = SpanCtxFromCtx(ctx)
				if parentSpanCtx != nil {
					spanRef = ot.FollowsFrom(parentSpanCtx)
				}
			}

			if parentSpanCtx == nil {
				err = next(ctx, spec, msg, cb)
				return
			}

			span := tracer.StartSpan(
				AsyncPublisherOpName(spec),
				spanRef,
				otext.SpanKindProducer,
				AsyncPublisherComponentTag,
			)
			fin := func(err error) {
				SetSpanError(span, err)
				span.Finish()
			}

			md := npmd.MDFromOutgoingContext(ctx)
			md, err = InjectSpanCtx(tracer, span.Context(), md)
			if err != nil {
				return
			}
			ctx = npmd.NewOutgoingContextWithMD(ctx, md)

			if err = next(ctx, spec, msg, func(cbErr error) {
				cb(cbErr)
				fin(cbErr)
			}); err != nil {
				fin(err)
			}
			return
		}
	}
}

// WrapMsgSubscriber adds opentracing support for MsgSubscriber.
func WrapMsgSubscriber(tracer ot.Tracer) MsgMiddleware {
	return func(spec MsgSpec, queue string, handler MsgHandler) MsgHandler {
		opName := SubscriberOpName(spec, queue)
		return func(ctx context.Context, msg interface{}) (err error) {
			parentSpanCtx, err := ExtractSpanCtx(
				tracer,
				npmd.MDFromIncomingContext(ctx),
			)
			if err != nil {
				return
			}

			if parentSpanCtx == nil {
				err = handler(ctx, msg)
				return
			}

			span := tracer.StartSpan(
				opName,
				ot.FollowsFrom(parentSpanCtx),
				otext.SpanKindConsumer,
				SubscriberComponentTag,
			)
			defer func() {
				SetSpanError(span, err)
				span.Finish()
			}()

			ctx = ot.ContextWithSpan(ctx, span)
			err = handler(ctx, msg)
			return

		}
	}
}
