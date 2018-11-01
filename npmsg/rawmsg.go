package npmsg

import (
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/huangjunwen/nproto"
	"github.com/huangjunwen/nproto/npmsg/enc"
)

// RawMsgPublisher is similar to MsgPublisher but operates on lower level.
type RawMsgPublisher interface {
	// Publish publishes a message to the given subject. It returns nil when succeeded.
	// When returning not nil error, it maybe succeeded or failed.
	Publish(ctx context.Context, subject string, data []byte) error
}

// RawBatchMsgPublisher can publish a batch of messages at once for higher throughput.
type RawBatchMsgPublisher interface {
	RawMsgPublisher

	// PublishBatch publishes a batch of messages.
	PublishBatch(ctx context.Context, subjects []string, datas [][]byte) (errors []error)
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

type defaultMsgSubscriber struct {
	encoder    enc.MsgSubscriberEncoder
	subscriber RawMsgSubscriber
}

var (
	_ nproto.MsgPublisher  = (*defaultMsgPublisher)(nil)
	_ nproto.MsgSubscriber = (*defaultMsgSubscriber)(nil)
)

// NewMsgPublisher creates a MsgPublisher from RawMsgPublisher and MsgPublisherEncoder.
func NewMsgPublisher(publisher RawMsgPublisher, encoder enc.MsgPublisherEncoder) nproto.MsgPublisher {
	return &defaultMsgPublisher{
		publisher: publisher,
		encoder:   encoder,
	}
}

// Publish implements MsgPublisher interface.
func (p *defaultMsgPublisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	// Encode payload.
	data, err := p.encoder.EncodePayload(&enc.MsgPayload{
		Msg:      msg,
		Passthru: nproto.Passthru(ctx),
	})
	if err != nil {
		panic(err)
	}

	// Publish.
	return p.publisher.Publish(ctx, subject, data)
}

// NewMsgSubscriber creates a MsgSubscriber from RawMsgSubscriber and MsgSubscriberEncoder.
func NewMsgSubscriber(subscriber RawMsgSubscriber, encoder enc.MsgSubscriberEncoder) nproto.MsgSubscriber {
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
		return handler(nproto.NewMsgCtx(ctx, subj, payload.Passthru), payload.Msg)
	}

	return s.subscriber.Subscribe(subject, queue, h, opts...)
}
