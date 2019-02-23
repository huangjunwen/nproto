package tracing

import (
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

type TracedMsgPublisher struct {
	publisher  nproto.MsgPublisher
	tracer     ot.Tracer
	downstream bool
}

type TraceMsgAsyncPublisher TracedMsgPublisher

type TracedMsgSubscriber struct {
	subscriber nproto.MsgSubscriber
	tracer     ot.Tracer
}

var (
	_ nproto.MsgPublisher      = (*TracedMsgPublisher)(nil)
	_ nproto.MsgAsyncPublisher = (*TraceMsgAsyncPublisher)(nil)
	_ nproto.MsgSubscriber     = (*TracedMsgSubscriber)(nil)
)

func NewTracedMsgPublisher(publisher nproto.MsgPublisher, tracer ot.Tracer) *TracedMsgPublisher {
	return &TracedMsgPublisher{
		publisher: publisher,
		tracer:    tracer,
	}
}

func NewDownstreamTracedMsgPublisher(publisher nproto.MsgPublisher, tracer ot.Tracer) *TracedMsgPublisher {
	return &TracedMsgPublisher{
		publisher:  publisher,
		tracer:     tracer,
		downstream: true,
	}
}

func NewTracedMsgAsyncPublisher(publisher nproto.MsgAsyncPublisher, tracer ot.Tracer) *TraceMsgAsyncPublisher {
	return &TraceMsgAsyncPublisher{
		publisher: publisher,
		tracer:    tracer,
	}
}

func NewDownstreamTracedMsgAsyncPublisher(publisher nproto.MsgAsyncPublisher, tracer ot.Tracer) *TraceMsgAsyncPublisher {
	return &TraceMsgAsyncPublisher{
		publisher:  publisher,
		tracer:     tracer,
		downstream: true,
	}
}

func (publisher *TracedMsgPublisher) Publish(ctx context.Context, subject string, msgData []byte) (err error) {

	tracer := publisher.tracer
	spanRef := ot.ChildOf(nil)

	// Gets parent span reference.
	if publisher.downstream {
		parentSpanCtx, err := extractSpanCtx(
			tracer,
			nproto.MDFromOutgoingContext(ctx),
			TracingMDKey,
		)
		if err != nil {
			return err
		}
		spanRef = ot.FollowsFrom(parentSpanCtx)

	} else {
		parentSpanCtx := spanCtxFromCtx(ctx)
		spanRef = ot.ChildOf(parentSpanCtx)

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

	// Injects span context. NOTE: `md[TracingMDKey]` is overwritten.
	md := nproto.MDFromOutgoingContext(ctx).Copy()
	err = injectSpanCtx(tracer, span.Context(), md, TracingMDKey)
	if err != nil {
		return
	}
	ctx = nproto.NewOutgoingContextWithMD(ctx, md)

	// Publish.
	err = publisher.publisher.Publish(ctx, subject, msgData)
	return
}

func (publisher *TraceMsgAsyncPublisher) Publish(ctx context.Context, subject string, msgData []byte) error {
	return (*TracedMsgPublisher)(publisher).Publish(ctx, subject, msgData)
}

func (publisher *TraceMsgAsyncPublisher) PublishAsync(ctx context.Context, subject string, msgData []byte, cb func(error)) (err error) {

	tracer := publisher.tracer
	spanRef := ot.ChildOf(nil)
	p := publisher.publisher.(nproto.MsgAsyncPublisher)

	// Gets parent span reference.
	if publisher.downstream {
		parentSpanCtx, err := extractSpanCtx(
			tracer,
			nproto.MDFromOutgoingContext(ctx),
			TracingMDKey,
		)
		if err != nil {
			return err
		}
		spanRef = ot.FollowsFrom(parentSpanCtx)

	} else {
		parentSpanCtx := spanCtxFromCtx(ctx)
		spanRef = ot.FollowsFrom(parentSpanCtx) // Use FollowFrom since PublishAsync is async op.

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

	// Injects span context. NOTE: `md[TracingMDKey]` is overwritten.
	md := nproto.MDFromOutgoingContext(ctx).Copy()
	err = injectSpanCtx(tracer, span.Context(), md, TracingMDKey)
	if err != nil {
		fin(err)
		return
	}
	ctx = nproto.NewOutgoingContextWithMD(ctx, md)

	// PublishAsync.
	if err = p.PublishAsync(ctx, subject, msgData, func(cbErr error) {
		fin(cbErr)
		cb(cbErr)
	}); err != nil {
		fin(err)
	}
	return
}

func NewTracedMsgSubscriber(subscriber nproto.MsgSubscriber, tracer ot.Tracer) *TracedMsgSubscriber {
	return &TracedMsgSubscriber{
		subscriber: subscriber,
		tracer:     tracer,
	}
}

func (subscriber *TracedMsgSubscriber) Subscribe(subject string, queue string, handler nproto.MsgHandler, opts ...interface{}) error {

	tracer := subscriber.tracer
	opName := SubscriberHandlerOpNameFmt(subject, queue)

	handler2 := func(ctx context.Context, msgData []byte) (err error) {
		// Gets parent span context.
		parentSpanCtx, err := extractSpanCtx(
			tracer,
			nproto.MDFromIncomingContext(ctx),
			TracingMDKey,
		)
		if err != nil {
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

	return subscriber.subscriber.Subscribe(subject, queue, handler2, opts...)
}