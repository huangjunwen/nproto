package npmsg

import (
	"context"
)

// Msg represents a message to be delivered reliably, e.g. at least once.
type Msg interface {
	// Subject is the subject/topic of the message.
	Subject() string

	// Data is the payload of the message.
	Data() []byte
}

// MsgPublisher is used to publish messages.
type MsgPublisher interface {
	// Publish a single message. It return nil if msg has been delivered successfully.
	Publish(msg Msg) error

	// PublishBatch publish a batch of messages. `len(errors) == len(msgs)` and
	// `errors[i]` is nil if `msgs[i]` has been delivered successfully.
	PublishBatch(msgs []Msg) (errors []error)
}

// MsgSubscriber is used to subscribe to subjects.
type MsgSubscriber interface {
	// Subscribe to a subject. Each message of the subject will be delivered to one
	// subscriber in the group. `opts` are implement specific options.
	Subscribe(subject, group string, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles Msg. The message should be re-deliver if the handler returns
// a not nil error.
type MsgHandler func(context.Context, Msg) error

// MsgMiddleware is used to decorate MsgHandler.
type MsgMiddleware func(MsgHandler) MsgHandler

type decMsgSubscriber struct {
	subscriber MsgSubscriber
	mws        []MsgMiddleware
}

// DecorateMsgSubscriber decorates a MsgSubscriber with MsgMiddlewares.
func DecorateMsgSubscriber(subscriber MsgSubscriber, mws ...MsgMiddleware) MsgSubscriber {
	return &decMsgSubscriber{
		subscriber: subscriber,
		mws:        mws,
	}
}

// Subscribe implements MsgSubscriber interface.
func (subscriber *decMsgSubscriber) Subscribe(subject, group string, handler MsgHandler, opts ...interface{}) error {
	// Decorate handler.
	for i := len(subscriber.mws) - 1; i >= 0; i-- {
		handler = subscriber.mws[i](handler)
	}
	return subscriber.subscriber.Subscribe(subject, group, handler, opts...)
}
