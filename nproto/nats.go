package nproto

import (
	"context"

	"github.com/nats-io/go-nats"
)

// NatsConn is an abstraction of nats.Conn
type NatsConn interface {
	PublishRequest(subj, reply string, data []byte) error
	RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error)
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error)
}

var (
	_ NatsConn = (*nats.Conn)(nil)
)
