package msg

import (
	"context"
)

// MsgPublisherFunc is an adapter to allow th use of ordinary functions as MsgPublisher.
type MsgPublisherFunc func(context.Context, string, interface{}) error

// MsgAsyncPublisherFunc is an adapter to allow the use of ordinary functions as MsgAsyncPublisher.
type MsgAsyncPublisherFunc func(context.Context, string, interface{}, func(error)) error

var (
	_ MsgPublisher      = (MsgPublisherFunc)(nil)
	_ MsgAsyncPublisher = (MsgAsyncPublisherFunc)(nil)
)

// Publish implements MsgPublisher interface.
func (fn MsgPublisherFunc) Publish(ctx context.Context, subject string, msg interface{}) error {
	return fn(ctx, subject, msg)
}

// Publish implements MsgAsyncPublisher interface.
func (fn MsgAsyncPublisherFunc) Publish(ctx context.Context, subject string, msg interface{}) error {
	var (
		err  error
		errc = make(chan struct{})
	)
	if err1 := fn(ctx, subject, msg, func(err2 error) {
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
func (fn MsgAsyncPublisherFunc) PublishAsync(ctx context.Context, subject string, msg interface{}, cb func(error)) error {
	return fn(ctx, subject, msg, cb)
}
