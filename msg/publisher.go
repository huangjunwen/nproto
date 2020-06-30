package msg

import (
	"context"
)

// MsgPublisher is used to publish messages reliably, e.g. at least once delivery.
type MsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil if success.
	Publish(ctx context.Context, spec *MsgSpec, msg interface{}) error
}

// MsgPublisherFunc is an adapter to allow the use of ordinary functions as MsgPublisher.
type MsgPublisherFunc func(context.Context, *MsgSpec, interface{}) error

// MsgPublisherMiddleware wraps MsgPublisher into another one.
type MsgPublisherMiddleware func(MsgPublisherFunc) MsgPublisherFunc

var (
	_ MsgPublisher = (MsgPublisherFunc)(nil)
)

// Publish implements MsgPublisher interface.
func (fn MsgPublisherFunc) Publish(ctx context.Context, spec *MsgSpec, msg interface{}) error {
	return fn(ctx, spec, msg)
}

// NewMsgPublisherWithMWs wraps a MsgPublisher with middlewares.
func NewMsgPublisherWithMWs(publisher MsgPublisher, mws ...MsgPublisherMiddleware) MsgPublisherFunc {
	p := publisher.Publish
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgPublisherFunc(p)
}
