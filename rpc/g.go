package librpc

import (
	"context"

	"github.com/huangjunwen/nproto/util"
	"github.com/nats-io/go-nats"
)

var (
	natsRequestWithContext = func(nc *nats.Conn, ctx context.Context, subj string, data []byte) (*nats.Msg, error) {
		return nc.RequestWithContext(ctx, subj, data)
	}
	natsPublish = func(nc *nats.Conn, subj string, data []byte) error {
		return nc.Publish(subj, data)
	}
	natsQueueSubscribe = func(nc *nats.Conn, subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
		return nc.QueueSubscribe(subj, queue, cb)
	}
	cfh util.ControlFlowHook = util.ProdControlFlowHook{}
)
