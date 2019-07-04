package nproto

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// MsgPublisher is used to publish messages reliably, e.g. at least once delivery.
type MsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil if success.
	Publish(ctx context.Context, subject string, msgData []byte) error
}

// MsgPublisherFunc is an adapter to allow th use of ordinary functions as MsgPublisher.
type MsgPublisherFunc func(context.Context, string, []byte) error

// MsgPublisherMiddleware wraps MsgPublisherFunc into another one.
type MsgPublisherMiddleware func(MsgPublisherFunc) MsgPublisherFunc

// MsgAsyncPublisher is similar to MsgPublisher but in async manner. It's trivial
// to implement MsgPublisher, see MsgAsyncPublisherFunc.
type MsgAsyncPublisher interface {
	// PublishAsync publishes a message to the given subject asynchronously.
	// The final result is returned by `cb` if PublishAsync returns nil.
	// `cb` must be called exactly once in this case.
	PublishAsync(ctx context.Context, subject string, msgData []byte, cb func(error)) error
}

// MsgAsyncPublisherFunc is an adapter to allow the use of ordinary functions as MsgAsyncPublisher.
type MsgAsyncPublisherFunc func(context.Context, string, []byte, func(error)) error

// MsgAsyncPublisherMiddleware wraps MsgAsyncPublisherFunc into another one.
type MsgAsyncPublisherMiddleware func(MsgAsyncPublisherFunc) MsgAsyncPublisherFunc

// MsgSubscriber is used to consume messages.
type MsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// In normal case (excpet message redelivery) each message will be delivered to
	// one member of each queue.
	// Order of messages is not guaranteed since redelivery.
	Subscribe(subject, queue string, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles messages. A message should be redelivered if the handler returns an error.
type MsgHandler func(context.Context, []byte) error

// MsgMiddleware wraps a MsgHandler into another one. The params are (subject, queue, handler).
type MsgMiddleware func(string, string, MsgHandler) MsgHandler

// MsgSubscriberWithMWs wraps a MsgSubscriber with middlewares.
type MsgSubscriberWithMWs struct {
	subscriber MsgSubscriber
	mws        []MsgMiddleware
}

// PBPublisher is used to publish protobuf message.
type PBPublisher struct {
	Publisher MsgPublisher
}

var (
	_ MsgPublisher      = (MsgPublisherFunc)(nil)
	_ MsgAsyncPublisher = (MsgAsyncPublisherFunc)(nil)
	_ MsgSubscriber     = (*MsgSubscriberWithMWs)(nil)
)

// Publish implements MsgPublisher interface.
func (fn MsgPublisherFunc) Publish(ctx context.Context, subject string, msgData []byte) error {
	return fn(ctx, subject, msgData)
}

// Publish implements MsgAsyncPublisher interface.
func (fn MsgAsyncPublisherFunc) Publish(ctx context.Context, subject string, msgData []byte) error {
	var (
		err  error
		errc = make(chan struct{})
	)
	if err1 := fn(ctx, subject, msgData, func(err2 error) {
		err = err2
		close(errc)
	}); err1 != nil {
		return err1
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-errc:
		return err
	}
}

// PublishAsync implements MsgAsyncPublisher interface.
func (fn MsgAsyncPublisherFunc) PublishAsync(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
	return fn(ctx, subject, msgData, cb)
}

// NewMsgPublisherWithMWs wraps a MsgPublisher with middlewares.
func NewMsgPublisherWithMWs(publisher MsgPublisher, mws ...MsgPublisherMiddleware) MsgPublisher {
	p := publisher.Publish
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgPublisherFunc(p)
}

// NewMsgAsyncPublisherWithMWs wraps a MsgAsyncPublisher with middlewares.
func NewMsgAsyncPublisherWithMWs(publisher MsgAsyncPublisher, mws ...MsgAsyncPublisherMiddleware) MsgAsyncPublisher {
	p := publisher.PublishAsync
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgAsyncPublisherFunc(p)
}

// Subscribe implements MsgSubscriber interface.
func (subscriber *MsgSubscriberWithMWs) Subscribe(subject, queue string, handler MsgHandler, opts ...interface{}) error {
	for i := len(subscriber.mws) - 1; i >= 0; i-- {
		handler = subscriber.mws[i](subject, queue, handler)
	}
	return subscriber.subscriber.Subscribe(subject, queue, handler, opts...)
}

// NewMsgSubscriberWithMWs creates a new MsgSubscriberWithMWs.
func NewMsgSubscriberWithMWs(subscriber MsgSubscriber, mws ...MsgMiddleware) *MsgSubscriberWithMWs {
	return &MsgSubscriberWithMWs{
		subscriber: subscriber,
		mws:        mws,
	}
}

// Publish publishes a protobuf message.
func (p *PBPublisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return p.Publisher.Publish(ctx, subject, msgData)
}
