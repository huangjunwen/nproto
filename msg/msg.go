package msg

import (
	"context"
)

// MsgPublisher is used to publish messages reliably, e.g. at least once delivery.
type MsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil if success.
	Publish(ctx context.Context, subject string, msg interface{}) error
}

// MsgAsyncPublisher is similar to MsgPublisher but in async manner. It's trivial
// to implement MsgPublisher, see MsgAsyncPublisherFunc.
type MsgAsyncPublisher interface {
	// PublishAsync publishes a message to the given subject asynchronously.
	// The final result is returned by `cb` if PublishAsync returns nil.
	// `cb` must be called exactly once in this case.
	PublishAsync(ctx context.Context, subject string, msg interface{}, cb func(error)) error
}

// MsgSubscriber is used to consume messages.
type MsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// In normal case (excpet message redelivery) each message will be delivered to
	// one member of each queue.
	//
	// Order of messages is not guaranteed since redelivery.
	Subscribe(subject, queue string, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles messages. A message should be redelivered if the handler returns an error.
type MsgHandler func(context.Context, interface{}) error

// MsgPublisherMiddleware wraps MsgPublisherFunc into another one.
type MsgPublisherMiddleware func(MsgPublisherFunc) MsgPublisherFunc

// MsgAsyncPublisherMiddleware wraps MsgAsyncPublisherFunc into another one.
type MsgAsyncPublisherMiddleware func(MsgAsyncPublisherFunc) MsgAsyncPublisherFunc

// MsgMiddleware wraps a MsgHandler into another one. The params are (subject, queue, handler).
type MsgMiddleware func(string, string, MsgHandler) MsgHandler
