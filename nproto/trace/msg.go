package trace

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/npmsg"
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

// TracedRawMsgPublisher wraps a RawMsgPublisher with an opentracing tracer.
type TracedRawMsgPublisher struct {
	publisher npmsg.RawMsgPublisher
	tracer    ot.Tracer
}

// TracedRawMsgAsyncPublisher wraps a RawMsgAsyncPublisher with an opentracing tracer.
type TracedRawMsgAsyncPublisher struct {
	publisher npmsg.RawMsgAsyncPublisher
	tracer    ot.Tracer
}

// TracedRawMsgSubscriber wraps a RawMsgSubscriber with an opentracing tracer.
type TracedRawMsgSubscriber struct {
	subscriber npmsg.RawMsgSubscriber
	tracer     ot.Tracer
}

var (
	_ nproto.MsgPublisher        = (*TracedMsgPublisher)(nil)
	_ nproto.MsgSubscriber       = (*TracedMsgSubscriber)(nil)
	_ npmsg.RawMsgPublisher      = (*TracedRawMsgPublisher)(nil)
	_ npmsg.RawMsgAsyncPublisher = (*TracedRawMsgAsyncPublisher)(nil)
	_ npmsg.RawMsgSubscriber     = (*TracedRawMsgSubscriber)(nil)
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
	// Starts pub span.
	ctx, pubSpan, err := startPubSpan(ctx, publisher.tracer, subject)
	if err != nil {
		return err
	}
	defer func() {
		// NOTE: Don't use
		//
		//   defer finishPubSpan(pubSpan, err)
		//
		// directly, since parameters of the defer function is evaluated at defer statement and can't be changed.
		finishPubSpan(pubSpan, err)
	}()

	// Publish.
	err = publisher.publisher.Publish(ctx, subject, msg)
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
		// Starts sub span.
		ctx, subSpan, err := startSubSpan(ctx, subscriber.tracer, subject, queue)
		if err != nil {
			return err
		}
		defer func() {
			finishSubSpan(subSpan, err)
		}()

		// Handle.
		err = handler(ctx, msg)
		return err

	}
	return subscriber.subscriber.Subscribe(subject, queue, newMsg, h, opts...)
}

// NewTracedRawMsgPublisher creates a new TracedRawMsgPublisher.
func NewTracedRawMsgPublisher(publisher npmsg.RawMsgPublisher, tracer ot.Tracer) *TracedRawMsgPublisher {
	return &TracedRawMsgPublisher{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish implements npmsg.RawMsgPublisher interface.
func (publisher *TracedRawMsgPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	// Starts pub span.
	ctx, pubSpan, err := startPubSpan(ctx, publisher.tracer, subject)
	if err != nil {
		return err
	}
	defer func() {
		finishPubSpan(pubSpan, err)
	}()

	// Publish.
	err = publisher.publisher.Publish(ctx, subject, data)
	return err
}

// NewTracedRawMsgAsyncPublisher creates a new TracedRawMsgAsyncPublisher.
func NewTracedRawMsgAsyncPublisher(publisher npmsg.RawMsgAsyncPublisher, tracer ot.Tracer) *TracedRawMsgAsyncPublisher {
	return &TracedRawMsgAsyncPublisher{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish implements npmsg.RawMsgAsyncPublisher interface.
func (publisher *TracedRawMsgAsyncPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	// Starts pub span.
	ctx, pubSpan, err := startPubSpan(ctx, publisher.tracer, subject)
	if err != nil {
		return err
	}
	defer func() {
		finishPubSpan(pubSpan, err)
	}()

	// Publish.
	err = publisher.publisher.Publish(ctx, subject, data)
	return err
}

// PublishAsync implements npmsg.RawMsgAsyncPublisher interface.
func (publisher *TracedRawMsgAsyncPublisher) PublishAsync(ctx context.Context, subject string, data []byte, cb func(error)) error {
	// Starts pub span.
	ctx, pubSpan, err := startPubSpan(ctx, publisher.tracer, subject)
	if err != nil {
		return err
	}

	// Publish.
	if err := publisher.publisher.PublishAsync(ctx, subject, data, func(cbErr error) {
		finishPubSpan(pubSpan, cbErr)
		cb(cbErr)
	}); err != nil {
		// If PublishAsync returns not nil error, cb must not be called.
		finishPubSpan(pubSpan, err)
		return err
	}
	return nil
}

// NewTracedRawMsgSubscriber creates a new TracedRawMsgSubscriber.
func NewTracedRawMsgSubscriber(subscriber npmsg.RawMsgSubscriber, tracer ot.Tracer) *TracedRawMsgSubscriber {
	return &TracedRawMsgSubscriber{
		subscriber: subscriber,
		tracer:     tracer,
	}
}

// Subscribe implements npmsg.RawMsgSubscriber interface.
func (subscriber *TracedRawMsgSubscriber) Subscribe(subject string, queue string, handler npmsg.RawMsgHandler, opts ...interface{}) error {
	h := func(ctx context.Context, subject string, data []byte) error {
		// Starts sub span.
		ctx, subSpan, err := startSubSpan(ctx, subscriber.tracer, subject, queue)
		if err != nil {
			return err
		}
		defer func() {
			finishSubSpan(subSpan, err)
		}()

		// Handle.
		err = handler(ctx, subject, data)
		return err
	}
	return subscriber.subscriber.Subscribe(subject, queue, h, opts...)
}

func startPubSpan(ctx context.Context, tracer ot.Tracer, subject string) (context.Context, ot.Span, error) {
	// Extract current span context, maybe nil.
	curSpanCtx := spanCtxFromCtx(ctx)

	// Start a publish span.
	pubSpan := tracer.StartSpan(
		subject,
		ot.ChildOf(curSpanCtx),
		otext.SpanKindProducer,
		msgComponentTag,
	)

	// Inject span context.
	md, err := injectSpanCtx(
		tracer,
		pubSpan.Context(),
		nproto.FromOutgoingContext(ctx),
	)
	if err != nil {
		return nil, nil, err
	}

	return nproto.NewOutgoingContext(ctx, md), pubSpan, nil
}

func finishPubSpan(pubSpan ot.Span, err error) {
	setSpanError(pubSpan, err)
	pubSpan.Finish()
}

func startSubSpan(ctx context.Context, tracer ot.Tracer, subject, queue string) (context.Context, ot.Span, error) {
	// Extract current span context, maybe nil.
	curSpanCtx, err := extractSpanCtx(
		tracer,
		nproto.CurrMsgMetaData(ctx),
	)
	if err != nil {
		return nil, nil, err
	}

	// Start a subscribe span.
	subSpan := tracer.StartSpan(
		fmt.Sprintf("%s:%s", subject, queue),
		ot.FollowsFrom(curSpanCtx),
		msgComponentTag,
		otext.SpanKindConsumer,
	)

	return ot.ContextWithSpan(ctx, subSpan), subSpan, nil
}

func finishSubSpan(subSpan ot.Span, err error) {
	setSpanError(subSpan, err)
	subSpan.Finish()
}
