package msg

// MsgSubscriberWithMWs wraps a MsgSubscriber with middlewares.
type MsgSubscriberWithMWs struct {
	subscriber MsgSubscriber
	mws        []MsgMiddleware
}

var (
	_ MsgSubscriber = (*MsgSubscriberWithMWs)(nil)
)

// NewMsgPublisherWithMWs wraps a MsgPublisher with middlewares.
func NewMsgPublisherWithMWs(publisher MsgPublisher, mws ...MsgPublisherMiddleware) MsgPublisher {
	p := publisher.Publish
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgPublisherFunc(p)
}

// NewMsgAsyncPublisherWithMWs wraps a MsgAsyncPublisher with middlewares.
func NewMsgAsyncPublisherWithMWs(publisher MsgAsyncPublisher, mws ...MsgAsyncPublisherMiddleware) MsgAsyncPublisher {
	p := publisher.PublishAsync
	for i := len(mws) - 1; i >= 0; i-- {
		p = mws[i](p)
	}
	return MsgAsyncPublisherFunc(p)
}

// NewMsgSubscriberWithMWs creates a new MsgSubscriberWithMWs.
func NewMsgSubscriberWithMWs(subscriber MsgSubscriber, mws ...MsgMiddleware) *MsgSubscriberWithMWs {
	return &MsgSubscriberWithMWs{
		subscriber: subscriber,
		mws:        mws,
	}
}

// Subscribe implements MsgSubscriber interface.
func (subscriber *MsgSubscriberWithMWs) Subscribe(subject, queue string, handler MsgHandler, opts ...interface{}) error {
	for i := len(subscriber.mws) - 1; i >= 0; i-- {
		handler = subscriber.mws[i](subject, queue, handler)
	}
	return subscriber.subscriber.Subscribe(subject, queue, handler, opts...)
}
