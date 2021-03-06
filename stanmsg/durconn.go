package stanmsg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/huangjunwen/golibs/taskrunner/limitedrunner"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/xid"
	"google.golang.org/protobuf/proto"

	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nppbmsg "github.com/huangjunwen/nproto/v2/pb/msg"
)

// DurConn is 'durable connection' to nats-streaming-server which handles reconnect and resubscription automatically.
type DurConn struct {
	// Immutable fields.
	nc                  *nats.Conn
	clusterID           string
	ctx                 context.Context
	subjectPrefix       string
	runner              taskrunner.TaskRunner // runner for handlers.
	logger              logr.Logger
	reconnectWait       time.Duration
	subRetryWait        time.Duration
	stanOptPingInterval int
	stanOptPingMaxOut   int
	stanOptPubAckWait   time.Duration
	connectCb           func(stan.Conn)
	disconnectCb        func(stan.Conn)
	subscribeCb         func(sc stan.Conn, spec MsgSpec)

	wg        sync.WaitGroup // wait for go routines
	connectMu sync.Mutex     // at most on connect can be run at any time
	mu        sync.RWMutex   // to protect mutable fields

	// Mutable fields.
	closed    bool
	subs      map[[2]string]*subscription // (subject, queue) -> subscriptioin
	sc        stan.Conn                   // nil if DurConn has not connected or is reconnecting
	scStaleCh chan struct{}               // pair with sc, it will closed when sc is stale
}

type subscription struct {
	spec        MsgSpec
	queue       string
	handler     MsgHandler
	stanOptions []stan.SubscriptionOption
	decoder     npenc.Decoder
}

// DurConnOption is option in creating DurConn.
type DurConnOption func(*DurConn) error

// SubOption is option in subscription.
type SubOption func(*subscription) error

// NewDurConn creates a new DurConn. `nc` must have MaxReconnect < 0
// (e.g. never give up trying to reconnect).
// `clusterID` is the nats-streaming-server's cluster id.
func NewDurConn(nc *nats.Conn, clusterID string, opts ...DurConnOption) (durConn *DurConn, err error) {
	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	dc := &DurConn{
		nc:                  nc,
		clusterID:           clusterID,
		ctx:                 context.Background(),
		subjectPrefix:       DefaultSubjectPrefix,
		runner:              limitedrunner.Must(),
		logger:              logr.Nop,
		reconnectWait:       DefaultReconnectWait,
		subRetryWait:        DefaultSubRetryWait,
		stanOptPingInterval: DefaultStanPingInterval,
		stanOptPingMaxOut:   DefaultStanPingMaxOut,
		stanOptPubAckWait:   DefaultStanPubAckWait,
		connectCb:           func(_ stan.Conn) {},
		disconnectCb:        func(_ stan.Conn) {},
		subscribeCb:         func(_ stan.Conn, _ MsgSpec) {},
		subs:                make(map[[2]string]*subscription),
	}

	defer func() {
		if err != nil {
			dc.runner.Close()
		}
	}()

	for _, opt := range opts {
		if err = opt(dc); err != nil {
			return nil, err
		}
	}

	dc.goConnect(false)
	return dc, nil
}

// NewPublisher creates a publisher using specified encoder.
func (dc *DurConn) NewPublisher(encoder npenc.Encoder) MsgAsyncPublisherFunc {
	return func(ctx context.Context, spec MsgSpec, msg interface{}, cb func(error)) error {
		return dc.publishAsync(ctx, spec, msg, encoder, cb)
	}
}

// NewSubscriber creates a subscriber using specified decoder.
func (dc *DurConn) NewSubscriber(decoder npenc.Decoder) MsgSubscriberFunc {
	return func(spec MsgSpec, queue string, handler MsgHandler, opts ...interface{}) error {
		sub := &subscription{
			spec:        spec,
			queue:       queue,
			handler:     handler,
			stanOptions: []stan.SubscriptionOption{},
			decoder:     decoder,
		}

		for _, opt := range opts {
			option, ok := opt.(SubOption)
			if !ok {
				return fmt.Errorf("Expect SubOption but got %v", opt)
			}
			if err := option(sub); err != nil {
				return err
			}
		}

		return dc.subscribeOne(sub)
	}
}

func (dc *DurConn) goConnect(wait bool) {
	dc.wg.Add(1)
	go func() {
		defer dc.wg.Done()
		dc.connect(wait)
	}()
}

func (dc *DurConn) connect(wait bool) {
	if wait {
		time.Sleep(dc.reconnectWait)
	}

	dc.connectMu.Lock()
	defer dc.connectMu.Unlock()

	// Reset connection: release old connection
	{
		dc.mu.Lock()
		if dc.closed {
			dc.mu.Unlock()
			dc.logger.Info("closed when reseting connection")
			return
		}
		sc := dc.sc
		scStaleCh := dc.scStaleCh
		dc.sc = nil
		dc.scStaleCh = nil
		dc.mu.Unlock()

		if sc != nil {
			sc.Close()
			close(scStaleCh)
		}
	}

	// Connect
	var sc stan.Conn
	{
		opts := []stan.Option{
			stan.Pings(dc.stanOptPingInterval, dc.stanOptPingMaxOut),
			stan.PubAckWait(dc.stanOptPubAckWait),
			stan.NatsConn(dc.nc),
			// NOTE: ConnectionLostHandler is used to be notified if the Streaming connection
			// is closed due to unexpected errors.
			// The callback will not be invoked on normal Conn.Close().
			stan.SetConnectionLostHandler(func(sc stan.Conn, err error) {
				dc.logger.Error(err, "connection lost")
				dc.disconnectCb(sc)
				// reconnect after a while
				dc.goConnect(true)
			}),
		}

		// NOTE: Use a UUID-like id as client id sine we only use durable queue subscription.
		// See: https://groups.google.com/d/msg/natsio/SkWAdSU1AgU/tCX9f3ONBQAJ
		var err error
		sc, err = stan.Connect(dc.clusterID, xid.New().String(), opts...)
		if err != nil {
			// reconnect immediately.
			dc.logger.Error(err, "connect failed")
			dc.goConnect(true)
			return
		}
		dc.logger.Info("connect successfully")
		dc.connectCb(sc)
	}

	// Update new connection
	var subs []*subscription
	scStaleCh := make(chan struct{})
	{
		dc.mu.Lock()
		if dc.closed {
			dc.mu.Unlock()
			sc.Close()
			close(scStaleCh)
			dc.logger.Info("closed when updating connection")
			return
		}
		dc.sc = sc
		dc.scStaleCh = scStaleCh
		for _, sub := range dc.subs {
			subs = append(subs, sub)
		}
		dc.mu.Unlock()
	}

	// Re-subscribe
	dc.goSubscribeAll(subs, sc, scStaleCh)

}

func (dc *DurConn) publishAsync(ctx context.Context, spec MsgSpec, msg interface{}, encoder npenc.Encoder, cb func(error)) error {

	if err := AssertMsgType(spec, msg); err != nil {
		return err
	}

	m := &nppbmsg.MessageWithMD{
		MetaData: nppbmd.NewMetaData(npmd.MDFromOutgoingContext(ctx)),
	}

	if err := encoder.EncodeData(msg, &m.MsgFormat, &m.MsgBytes); err != nil {
		return err
	}

	mData, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	dc.mu.RLock()
	closed := dc.closed
	sc := dc.sc
	dc.mu.RUnlock()

	if closed {
		return ErrClosed
	}
	if sc == nil {
		return ErrNotConnected
	}

	// Publish.
	// TODO: sc.PublishAsync maybe block in some rare condition:
	// see https://github.com/nats-io/stan.go/issues/210
	_, err = sc.PublishAsync(
		subjectFormat(dc.subjectPrefix, spec.SubjectName()),
		mData,
		func(_ string, err error) { cb(err) },
	)
	return err

}

func (dc *DurConn) subscribeOne(sub *subscription) error {
	key := [2]string{sub.spec.SubjectName(), sub.queue}
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.closed {
		return ErrClosed
	}

	if _, ok := dc.subs[key]; ok {
		return ErrDupSubscription
	}

	// subscribe if sc is not nil
	if dc.sc != nil {
		if err := dc.subscribe(sub, dc.sc); err != nil {
			return err
		}
	}

	dc.subs[key] = sub
	return nil
}

func (dc *DurConn) goSubscribeAll(subs []*subscription, sc stan.Conn, scStaleCh chan struct{}) {
	dc.wg.Add(1)
	go func() {
		defer dc.wg.Done()
		dc.subscribeAll(subs, sc, scStaleCh)
	}()
}

// NOTE: subscribe until all success or scStaleCh closed.
func (dc *DurConn) subscribeAll(subs []*subscription, sc stan.Conn, scStaleCh chan struct{}) {

	success := make([]bool, len(subs))
	for {
		n := 0
		for i, sub := range subs {
			if success[i] {
				// Already success.
				n++
				continue
			}
			if err := dc.subscribe(sub, sc); err != nil {
				continue
			}
			success[i] = true
			n++

			select {
			case <-scStaleCh:
				dc.logger.Info("subscribe stale")
				return
			default:
			}
		}

		if n >= len(subs) {
			// All success.
			return
		}

		select {
		case <-scStaleCh:
			dc.logger.Info("subscribe stale during retry wait")
			return

		case <-time.After(dc.subRetryWait):
		}

	}

}

func (dc *DurConn) subscribe(sub *subscription, sc stan.Conn) error {

	fullSubject := subjectFormat(dc.subjectPrefix, sub.spec.SubjectName())
	opts := []stan.SubscriptionOption{}
	opts = append(opts, sub.stanOptions...)
	opts = append(opts, stan.SetManualAckMode())     // Use manual ack mode.
	opts = append(opts, stan.DurableName(sub.queue)) // Queue as durable name.
	_, err := sc.QueueSubscribe(fullSubject, sub.queue, dc.msgHandler(sub), opts...)
	if err != nil {
		dc.logger.Error(err, "subscribe error", "subject", sub.spec.SubjectName(), "queue", sub.queue)
		return err
	}
	dc.logger.Info("subscribe successfully", "subject", sub.spec.SubjectName(), "queue", sub.queue)
	dc.subscribeCb(sc, sub.spec)
	return nil

}

func (dc *DurConn) msgHandler(sub *subscription) stan.MsgHandler {

	logger := dc.logger.WithValues("subject", sub.spec.SubjectName(), "queue", sub.queue)

	return func(stanMsg *stan.Msg) {

		if err := dc.runner.Submit(func() {
			defer func() {
				if e := recover(); e != nil {
					err, ok := e.(error)
					if !ok {
						err = fmt.Errorf("%+v", e)
					}
					logger.Error(err, "handler panic")
				}
			}()

			m := &nppbmsg.MessageWithMD{}
			if err := proto.Unmarshal(stanMsg.Data, m); err != nil {
				logger.Error(err, "unmarshal msg error", "data", stanMsg.Data)
				return
			}

			msg := sub.spec.NewMsg()
			if err := sub.decoder.DecodeData(m.MsgFormat, m.MsgBytes, msg); err != nil {
				logger.Error(err, "decode msg error")
				return
			}

			ctx := dc.ctx
			if len(m.MetaData) != 0 {
				ctx = npmd.NewIncomingContextWithMD(ctx, nppbmd.MetaData(m.MetaData))
			}

			if err := sub.handler(ctx, msg); err != nil {
				// NOTE: do not print handle's error log. Let the handler do it itself.
				return
			}

			// Ack if no error.
			stanMsg.Ack()
			return

		}); err != nil {

			logger.Error(err, "submit task error")

		}

	}

}

// Close shutdowns the DurConn: closes handler runner and disconnects from nats-streaming-server.
func (dc *DurConn) Close() {
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return
	}
	sc := dc.sc
	scStaleCh := dc.scStaleCh
	dc.sc = nil
	dc.scStaleCh = nil
	dc.closed = true
	dc.mu.Unlock()

	if sc != nil {
		sc.Close()
		close(scStaleCh)
	}

	dc.runner.Close()
	dc.wg.Wait()

}
