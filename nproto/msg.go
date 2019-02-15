package nproto

import (
	"context"
)

// MsgPublisher is used to publish messages reliably, e.g. at least once delivery.
type MsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil if success.
	// MetaData attached to `ctx` must be passed unmodified to downstream:
	// downstream publishers or subscribed handlers.
	Publish(ctx context.Context, subject string, data []byte) error
}

// MsgAsyncPublisher extends MsgPublisher with PublishAsync.
type MsgAsyncPublisher interface {
	// It's trivial to implement Publish if it supports PublishAsync. See MsgAsyncPublisherFunc.
	MsgPublisher

	// PublishAsync publishes a message to the given subject asynchronously.
	// The final result is returned by `cb` if PublishAsync returns nil.
	// `cb` must be called exactly once in this case.
	// MetaData attached to `ctx` must be passed unmodified to downstream:
	// downstream publishers or subscribed handlers.
	PublishAsync(ctx context.Context, subject string, data []byte, cb func(error)) error
}

// MsgAsyncPublisherFunc is an adapter to allow the use of ordinary functions as MsgAsyncPublisher.
type MsgAsyncPublisherFunc func(context.Context, string, []byte, func(error)) error

// MsgSubscriber is used to consume messages.
type MsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// In normal case (excpet message redelivery) each message will be delivered to
	// one member of each queue.
	Subscribe(subject, queue string, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles messages. A message should be redelivered if the handler returns an error.
type MsgHandler func(context.Context, []byte) error

var (
	_ MsgAsyncPublisher = (MsgAsyncPublisherFunc)(nil)
)

// Publish implements MsgAsyncPublisher interface.
func (fn MsgAsyncPublisherFunc) Publish(ctx context.Context, subject string, data []byte) error {
	var (
		err  error
		errc = make(chan struct{})
	)
	if err1 := fn(ctx, subject, data, func(err2 error) {
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
func (fn MsgAsyncPublisherFunc) PublishAsync(ctx context.Context, subject string, data []byte, cb func(error)) error {
	return fn(ctx, subject, data, cb)
}
