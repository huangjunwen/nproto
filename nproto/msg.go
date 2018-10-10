package nproto

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// MsgPublisher is used to publish messages reliably, e.g. at least once delivery.
type MsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil when succeeded.
	Publish(ctx context.Context, subject string, msg proto.Message) error
}

// MsgSubscriber is used to consume messages.
type MsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// In normal case (excpet message redelivery) each message will be delivered to
	// one member of each queue.
	Subscribe(subject, queue string, newMsg func() proto.Message, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles the message. The message should be redelivered if it returns an error.
type MsgHandler func(context.Context, proto.Message) error

// MsgMiddleware is used to decorate MsgHandler.
type MsgMiddleware func(MsgHandler) MsgHandler

type decMsgSubscriber struct {
	MsgSubscriber
	mws []MsgMiddleware
}

// DecorateMsgSubscriber decorates a MsgSubscriber with MsgMiddlewares.
func DecorateMsgSubscriber(subscriber MsgSubscriber, mws ...MsgMiddleware) MsgSubscriber {
	return &decMsgSubscriber{
		MsgSubscriber: subscriber,
		mws:           mws,
	}
}

// Subscribe implements MsgSubscriber interface.
func (subscriber *decMsgSubscriber) Subscribe(subject, queue string, newMsg func() proto.Message, handler MsgHandler, opts ...interface{}) error {
	// Decorate handler.
	for i := len(subscriber.mws) - 1; i >= 0; i-- {
		handler = subscriber.mws[i](handler)
	}
	return subscriber.MsgSubscriber.Subscribe(subject, queue, newMsg, handler, opts...)
}
