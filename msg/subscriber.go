package msg

import (
	"context"
)

// MsgSubscriber is used to consume messages.
type MsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// In normal case (excpet message redelivery) each message will be delivered to
	// one member of each queue.
	//
	// Order of messages is not guaranteed since redelivery.
	Subscribe(spec MsgSpec, queue string, handler MsgHandler, opts ...interface{}) error
}

// MsgHandler handles messages. For 'at least once delivery' implementations, a message
// should be redelivered if the handler returns an error. Otherwise it may or may not
// be redelivered.
type MsgHandler func(context.Context, interface{}) error

// MsgSubscriberFunc is an adapter to allow the use of ordinary functions as MsgSubscriber.
type MsgSubscriberFunc func(MsgSpec, string, MsgHandler, ...interface{}) error

// MsgMiddleware wraps a MsgHandler into another one.
type MsgMiddleware func(spec MsgSpec, queue string, handler MsgHandler) MsgHandler

// MsgSubscriberWithMWs wraps a MsgSubscriber with middlewares.
type MsgSubscriberWithMWs struct {
	subscriber MsgSubscriber
	mws        []MsgMiddleware
}

var (
	_ MsgSubscriber = (MsgSubscriberFunc)(nil)
	_ MsgSubscriber = (*MsgSubscriberWithMWs)(nil)
)

// Subscribe implements MsgSubscriber interface.
func (fn MsgSubscriberFunc) Subscribe(spec MsgSpec, queue string, handler MsgHandler, opts ...interface{}) error {
	return fn(spec, queue, handler, opts...)
}

// NewMsgSubscriberWithMWs creates a new MsgSubscriberWithMWs.
func NewMsgSubscriberWithMWs(subscriber MsgSubscriber, mws ...MsgMiddleware) *MsgSubscriberWithMWs {
	return &MsgSubscriberWithMWs{
		subscriber: subscriber,
		mws:        mws,
	}
}

// Subscribe implements MsgSubscriber interface.
func (subscriber *MsgSubscriberWithMWs) Subscribe(spec MsgSpec, queue string, handler MsgHandler, opts ...interface{}) error {
	for i := len(subscriber.mws) - 1; i >= 0; i-- {
		handler = subscriber.mws[i](spec, queue, handler)
	}
	return subscriber.subscriber.Subscribe(spec, queue, handler, opts...)
}
