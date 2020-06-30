package msg

import (
	"context"
)

// MsgAsyncPublisher is similar to MsgPublisher but in async manner. It's trivial
// to implement MsgPublisher, see MsgAsyncPublisherFunc.
type MsgAsyncPublisher interface {
	// PublishAsync publishes a message to the given subject asynchronously.
	// The final result is returned by `cb` if PublishAsync returns nil.
	// `cb` must be called exactly once in this case.
	PublishAsync(ctx context.Context, spec *MsgSpec, msg interface{}, cb func(error)) error
}

// MsgAsyncPublisherFunc is an adapter to allow the use of ordinary functions as MsgAsyncPublisher.
type MsgAsyncPublisherFunc func(context.Context, *MsgSpec, interface{}, func(error)) error

// MsgAsyncPublisherMiddleware wraps MsgAsyncPublisher into another one.
type MsgAsyncPublisherMiddleware func(MsgAsyncPublisherFunc) MsgAsyncPublisherFunc

var (
	_ MsgPublisher      = (MsgAsyncPublisherFunc)(nil)
	_ MsgAsyncPublisher = (MsgAsyncPublisherFunc)(nil)
)

// Publish implements MsgAsyncPublisher interface.
func (fn MsgAsyncPublisherFunc) Publish(ctx context.Context, spec *MsgSpec, msg interface{}) error {
	var (
		err  error
		errc = make(chan struct{})
	)
	if err1 := fn(ctx, spec, msg, func(err2 error) {
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
func (fn MsgAsyncPublisherFunc) PublishAsync(ctx context.Context, spec *MsgSpec, msg interface{}, cb func(error)) error {
	return fn(ctx, spec, msg, cb)
}

// NewMsgAsyncPublisherWithMWs wraps a MsgAsyncPublisher with middlewares.
func NewMsgAsyncPublisherWithMWs(publisher MsgAsyncPublisher, mws ...MsgAsyncPublisherMiddleware) MsgAsyncPublisherFunc {
	p := publisher.PublishAsync
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgAsyncPublisherFunc(p)
}
