package trace

import (
	"context"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

var (
	msgComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nproto.msg",
	}
)

// TracedMsgPublisher wraps a MsgPublisher with an opentracing tracer.
type TracedMsgPublisher struct {
	publisher nproto.MsgPublisher
	tracer    ot.Tracer
}

// TracedMsgSubscriber wraps a MsgSubscriber with an opentracing tracer.
type TracedMsgSubscriber struct {
	subscriber nproto.MsgSubscriber
	tracer     ot.Tracer
}

var (
	_ nproto.MsgPublisher  = (*TracedMsgPublisher)(nil)
	_ nproto.MsgSubscriber = (*TracedMsgSubscriber)(nil)
)

// NewTracedMsgPublisher creates a new TracedMsgPublisher.
func NewTracedMsgPublisher(publisher nproto.MsgPublisher, tracer ot.Tracer) *TracedMsgPublisher {
	return &TracedMsgPublisher{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish implements nproto.MsgPublisher interface.
func (publisher *TracedMsgPublisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	// Extract current span context, maybe nil.
	curSpanCtx := spanCtxFromCtx(ctx)

	// Start a publish span.
	pubSpan := publisher.tracer.StartSpan(
		subject,
		ot.ChildOf(curSpanCtx),
		otext.SpanKindProducer,
		msgComponentTag,
	)
	defer pubSpan.Finish()

	// Inject span context.
	md, err := injectSpanCtx(
		publisher.tracer,
		pubSpan.Context(),
		nproto.FromOutgoingContext(ctx),
	)
	if err != nil {
		return err
	}
	ctx = nproto.NewOutgoingContext(ctx, md)

	// Publish.
	err = publisher.publisher.Publish(ctx, subject, msg)
	setSpanError(pubSpan, err)
	return err
}

// NewTracedMsgSubscriber creates a new TracedMsgSubscriber.
func NewTracedMsgSubscriber(subscriber nproto.MsgSubscriber, tracer ot.Tracer) *TracedMsgSubscriber {
	return &TracedMsgSubscriber{
		subscriber: subscriber,
		tracer:     tracer,
	}
}

// Subscribe implements nproto.MsgSubscriber interface.
func (subscriber *TracedMsgSubscriber) Subscribe(subject, queue string, newMsg func() proto.Message, handler nproto.MsgHandler, opts ...interface{}) error {
	h := func(ctx context.Context, msg proto.Message) error {
		// Extract current span context, maybe nil.
		curSpanCtx, err := extractSpanCtx(
			subscriber.tracer,
			nproto.CurrMsgMetaData(ctx),
		)
		if err != nil {
			return err
		}

		// Start a subscribe span.
		subSpan := subscriber.tracer.StartSpan(
			subject,
			ot.FollowsFrom(curSpanCtx),
			msgComponentTag,
			otext.SpanKindConsumer,
		)
		defer subSpan.Finish()
		ctx = ot.ContextWithSpan(ctx, subSpan)

		// Handle.
		err = handler(ctx, msg)
		setSpanError(subSpan, err)
		return err

	}
	return subscriber.subscriber.Subscribe(subject, queue, newMsg, h, opts...)
}
