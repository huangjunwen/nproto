package npmsg

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/npmsg/enc"
)

// RawMsgPublisher is similar to MsgPublisher but operates on lower level.
type RawMsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil when succeeded.
	// When returning not nil error, it maybe succeeded or failed.
	Publish(ctx context.Context, subject string, data []byte) error
}

// RawMsgAsyncPublisher is similar to MsgAsyncPublisher but operates on lower level.
type RawMsgAsyncPublisher interface {
	// PublishAsync publishes a message to the given subject asynchronously.
	// The final result is returned by cb.
	// NOTE: This method must be non-blocking.
	// And cb must be called exactly once (even after context done) if PublishAsync returns nil.
	PublishAsync(ctx context.Context, subject string, data []byte, cb func(error)) error
}

// RawMsgSubscriber is similar to MsgSubscriber but operates on lower level.
type RawMsgSubscriber interface {
	// Subscribe subscribes to a given subject. One subject can have many queues.
	// A message will be delivered to one subscription of each queue.
	Subscribe(subject, queue string, handler RawMsgHandler, opts ...interface{}) error
}

// RawMsgHandler handles the message. The message should be redelivered if it returns an error.
type RawMsgHandler func(context.Context, string, []byte) error

type defaultMsgPublisher struct {
	encoder   enc.MsgPublisherEncoder
	publisher RawMsgPublisher
}

type defaultMsgAsyncPublisher struct {
	encoder   enc.MsgPublisherEncoder
	publisher RawMsgAsyncPublisher
}

type defaultMsgSubscriber struct {
	encoder    enc.MsgSubscriberEncoder
	subscriber RawMsgSubscriber
}

var (
	_ nproto.MsgPublisher      = (*defaultMsgPublisher)(nil)
	_ nproto.MsgAsyncPublisher = (*defaultMsgAsyncPublisher)(nil)
	_ nproto.MsgSubscriber     = (*defaultMsgSubscriber)(nil)
)

// NewMsgPublisher creates a MsgPublisher from RawMsgPublisher and MsgPublisherEncoder.
func NewMsgPublisher(publisher RawMsgPublisher, encoder enc.MsgPublisherEncoder) nproto.MsgPublisher {
	if encoder == nil {
		encoder = enc.PBPublisherEncoder{}
	}
	return &defaultMsgPublisher{
		publisher: publisher,
		encoder:   encoder,
	}
}

// Publish implements MsgPublisher interface.
func (p *defaultMsgPublisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	data, err := p.encoder.EncodePayload(&enc.MsgPayload{
		Msg:      msg,
		MetaData: nproto.FromOutgoingContext(ctx),
	})
	if err != nil {
		panic(err)
	}
	return p.publisher.Publish(ctx, subject, data)
}

// NewMsgAsyncPublisher creates a MsgAsyncPublisher from RawMsgAsyncPublisher and MsgPublisherEncoder.
func NewMsgAsyncPublisher(publisher RawMsgAsyncPublisher, encoder enc.MsgPublisherEncoder) nproto.MsgAsyncPublisher {
	if encoder == nil {
		encoder = enc.PBPublisherEncoder{}
	}
	return &defaultMsgAsyncPublisher{
		publisher: publisher,
		encoder:   encoder,
	}
}

// PublishAsync implements MsgAsyncPublisher interface.
func (p *defaultMsgAsyncPublisher) PublishAsync(ctx context.Context, subject string, msg proto.Message, cb func(error)) error {
	data, err := p.encoder.EncodePayload(&enc.MsgPayload{
		Msg:      msg,
		MetaData: nproto.FromOutgoingContext(ctx),
	})
	if err != nil {
		panic(err)
	}
	return p.publisher.PublishAsync(ctx, subject, data, cb)
}

// NewMsgSubscriber creates a MsgSubscriber from RawMsgSubscriber and MsgSubscriberEncoder.
func NewMsgSubscriber(subscriber RawMsgSubscriber, encoder enc.MsgSubscriberEncoder) nproto.MsgSubscriber {
	if encoder == nil {
		encoder = enc.PBSubscriberEncoder{}
	}
	return &defaultMsgSubscriber{
		subscriber: subscriber,
		encoder:    encoder,
	}
}

// Subscribe implements MsgSubscriber interface.
func (s *defaultMsgSubscriber) Subscribe(subject, queue string, newMsg func() proto.Message, handler nproto.MsgHandler, opts ...interface{}) error {

	h := func(ctx context.Context, subj string, data []byte) error {
		// Decode payload.
		payload := &enc.MsgPayload{
			// Init with an empty message so that the encoder can get type information of msg.
			Msg: newMsg(),
		}
		if err := s.encoder.DecodePayload(data, payload); err != nil {
			return err
		}

		// Setup context and run the handler.
		return handler(nproto.NewMsgCtx(ctx, subj, payload.MetaData), payload.Msg)
	}

	return s.subscriber.Subscribe(subject, queue, h, opts...)
}

// SyncPublish publish a message in sync manner using a RawMsgAsyncPublisher.
func SyncPublish(p RawMsgAsyncPublisher, ctx context.Context, subject string, data []byte) error {
	var (
		err  error
		errc = make(chan struct{})
	)
	if e1 := p.PublishAsync(ctx, subject, data, func(err2 error) {
		err = err2
		close(errc)
	}); e1 != nil {
		return e1
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-errc:
		return err
	}
}
