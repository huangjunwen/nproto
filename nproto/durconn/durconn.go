package durconn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/rs/xid"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/helpers/taskrunner"
	"github.com/huangjunwen/nproto/nproto"
	enc "github.com/huangjunwen/nproto/nproto/msgenc"
	"github.com/huangjunwen/nproto/nproto/zlog"
)

var (
	// DefaultSubjectPrefix is the default value of OptSubjectPrefix.
	DefaultSubjectPrefix = "npmsg"
	// DefaultReconnectWait is the default value of OptReconnectWait.
	DefaultReconnectWait = 5 * time.Second
	// DefaultSubRetryWait is the default value of SubOptRetryWait.
	DefaultSubRetryWait = 5 * time.Second
	// DefaultPubAckWait is the default value of OptPubAckWait.
	DefaultPubAckWait = 2 * time.Second
)

var (
	// ErrNCMaxReconnect is returned if nc has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("nproto.durconn.NewDurConn: nats.Conn should have MaxReconnects < 0")
	// ErrClosed is returned when the DurConn has been closed.
	ErrClosed = errors.New("nproto.durconn.DurConn: Closed.")
	// ErrNotConnected is returned when the underly connection has not been established.
	ErrNotConnected = errors.New("nproto.durconn.DurConn: Not connected.")
	// ErrBadSubscriptionOpt is returned if option passing to DurConn.Subscribe is not SubOption.
	ErrBadSubscriptionOpt = errors.New("nproto.durconn.DurConn: Expect durconn.SubOption.")
	// ErrDupSubscription is returned when duplicated (subject, queue) pair.
	ErrDupSubscription = func(subject, queue string) error {
		return fmt.Errorf(
			"nproto.durconn.DurConn: Duplicated subscription (%q, %q)",
			subject,
			queue,
		)
	}
)

var (
	_ nproto.MsgAsyncPublisher = (*DurConn)(nil)
	_ nproto.MsgPublisher      = (*DurConn)(nil)
	_ nproto.MsgSubscriber     = (*DurConn)(nil)
)

// DurConn implements nproto.MsgAsyncPublisher and nproto.MsgSubscriber interfaces.
type DurConn struct {
	// Options.
	logger        zerolog.Logger
	reconnectWait time.Duration
	subjectPrefix string
	encoder       enc.MsgPayloadEncoder
	decoder       enc.MsgPayloadDecoder
	connectCb     func(stan.Conn)
	disconnectCb  func(stan.Conn)
	hRunner       taskrunner.TaskRunner // used to run handlers.
	sRunner       taskrunner.TaskRunner // used to run subscription.
	stanOptions   []stan.Option

	// Immutable fields.
	clusterID string
	nc        *nats.Conn
	ctx       context.Context

	// At most one connect can be run at any time.
	connectMu sync.Mutex

	// Mutable fields.
	// mu is used to protect the following fields.
	mu       sync.RWMutex
	closed   bool              // true if the DurConn is closed.
	sc       stan.Conn         // nil if not connected/reconnecting.
	stalec   chan struct{}     // stalec is pair with sc, it is closed when sc is stale.
	subs     []*subscription   // list of subscription, append-only (since no Unsubscribe)
	subNames map[[2]string]int // (subject, queue) -> subscription index
}

type subscription struct {
	// Options.
	stanOptions []stan.SubscriptionOption
	retryWait   time.Duration
	subscribeCb func(sc stan.Conn, subject, queue string)

	// Immutable fields.
	dc      *DurConn
	subject string
	queue   string
	handler nproto.MsgHandler
}

// Option is option in creating DurConn.
type Option func(*DurConn) error

// SubOption is option in subscription.
type SubOption func(*subscription) error

// NewDurConn creates a new DurConn. `nc` must have MaxReconnect < 0 set (e.g. Always reconnect).
// `clusterID` is nats-streaming server's cluster id.
func NewDurConn(nc *nats.Conn, clusterID string, opts ...Option) (*DurConn, error) {
	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	dc := &DurConn{
		logger:        zerolog.Nop(),
		reconnectWait: DefaultReconnectWait,
		subjectPrefix: DefaultSubjectPrefix,
		encoder:       enc.PBMsgPayloadEncoder{}, // TODO: option
		decoder:       enc.PBMsgPayloadDecoder{}, // TODO: option
		connectCb:     func(_ stan.Conn) {},
		disconnectCb:  func(_ stan.Conn) {},
		hRunner:       taskrunner.NewDefaultLimitedRunner(),
		sRunner:       taskrunner.NewLimitedRunner(10, -1),
		stanOptions: []stan.Option{
			stan.PubAckWait(DefaultPubAckWait),
		},
		clusterID: clusterID,
		nc:        nc,
		ctx:       context.Background(),
		subNames:  make(map[[2]string]int),
	}
	OptLogger(&zlog.DefaultZLogger)(dc)

	for _, opt := range opts {
		if err := opt(dc); err != nil {
			return nil, err
		}
	}

	dc.connect(false)
	return dc, nil

}

// Graceful close the conn.
func (dc *DurConn) Close() error {
	// Stops the runner.
	dc.hRunner.Close()

	// Close connection.
	oldSc, oldStalec, err := dc.close_()
	if oldSc != nil {
		oldSc.Close()
		close(oldStalec)
	}
	return err
}

// PublishAsync implements nproto.MsgAsyncPublisher interface.
func (dc *DurConn) PublishAsync(ctx context.Context, subject string, msgData []byte, cb func(error)) error {
	// Encode payload.
	data, err := dc.encoder.EncodePayload(&enc.MsgPayload{
		MsgData: msgData,
		MD:      nproto.MDFromOutgoingContext(ctx),
	})
	if err != nil {
		return err
	}

	// Check status and get connection.
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
		fmt.Sprintf("%s.%s", dc.subjectPrefix, subject),
		data,
		func(_ string, err error) { cb(err) },
	)
	return err
}

// Publish implements nproto.MsgPublisher interface.
func (dc *DurConn) Publish(ctx context.Context, subject string, msgData []byte) error {
	return nproto.MsgAsyncPublisherFunc(dc.PublishAsync).Publish(ctx, subject, msgData)
}

// Subscribe implements nproto.MsgSubscriber interface. `opts` must be SubOption.
func (dc *DurConn) Subscribe(subject, queue string, handler nproto.MsgHandler, opts ...interface{}) error {

	sub := &subscription{
		retryWait:   DefaultSubRetryWait,
		subscribeCb: func(_ stan.Conn, _, _ string) {},
		dc:          dc,
		subject:     subject,
		queue:       queue,
		handler:     handler,
	}
	for _, opt := range opts {
		option, ok := opt.(SubOption)
		if !ok {
			return ErrBadSubscriptionOpt
		}
		if err := option(sub); err != nil {
			return err
		}
	}

	// Add to subscription list and subscribe.
	sc, stalec, err := dc.addSub(sub)
	if err != nil {
		return err
	}

	if sc != nil {
		dc.subscribe(sub, sc, stalec)
	}
	return nil
}

// connect is used to make a new stan connection.
func (dc *DurConn) connect(wait bool) {
	go func() {
		if wait {
			time.Sleep(dc.reconnectWait)
		}

		dc.connectMu.Lock()
		defer dc.connectMu.Unlock()

		// Release old sc and reset to nil.
		{
			oldSc, oldStalec, _, err := dc.reset(nil, nil)
			if oldSc != nil {
				oldSc.Close()
				close(oldStalec)
			}
			if err != nil {
				if err == ErrClosed {
					dc.logger.Info().Msg("DurConn closed when reseting connection")
					return
				}
				panic(err)
			}
		}

		// Connect.
		opts := []stan.Option{}
		opts = append(opts, dc.stanOptions...)
		opts = append(opts, stan.NatsConn(dc.nc))
		// Reconnect when connection lost. NOTE: This callback will not be called in
		// normal close.
		opts = append(opts, stan.SetConnectionLostHandler(func(sc stan.Conn, _ error) {
			dc.disconnectCb(sc)
			dc.connect(true)
		}))

		// NOTE: Use a UUID-like id as client id sine we only use durable queue subscription.
		// See: https://groups.google.com/d/msg/natsio/SkWAdSU1AgU/tCX9f3ONBQAJ
		sc, err := stan.Connect(dc.clusterID, xid.New().String(), opts...)
		if err != nil {
			// Reconnect immediately.
			dc.logger.Error().Err(err).Msg("Connect failed")
			dc.connect(true)
			return
		}
		dc.logger.Info().Msg("Connected")
		dc.connectCb(sc)

		// Update to the new connection.
		{
			stalec := make(chan struct{})
			_, _, subs, err := dc.reset(sc, stalec)
			if err != nil {
				sc.Close() // Release the new one.
				close(stalec)
				if err == ErrClosed {
					dc.logger.Info().Msg("DurConn closed when updating connection")
					return
				}
				panic(err)
			}

			// Re-subscribe.
			for _, sub := range subs {
				dc.subscribe(sub, sc, stalec)
			}
		}

		return
	}()

}

// subscribe keeps subscribing unless success or the stan connection is stale
// (e.g. disconnected or closed).
func (dc *DurConn) subscribe(sub *subscription, sc stan.Conn, stalec chan struct{}) {

	fullSubject := fmt.Sprintf("%s.%s", dc.subjectPrefix, sub.subject)

	// Wrap handler.
	handler := func(m *stan.Msg) {
		// Use task runner to run the handler.
		if err := dc.hRunner.Submit(func() {
			// Check subject.
			if m.Subject != fullSubject {
				panic(fmt.Sprintf("Expect stan subject %+q but got %+q", fullSubject, m.Subject))
			}

			// Decode payload.
			payload := &enc.MsgPayload{}
			if err := dc.decoder.DecodePayload(m.Data, payload); err != nil {
				dc.logger.Error().Err(err).Msg("Decode error")
				return
			}

			// Setup context.
			ctx := dc.ctx
			if payload.MD != nil {
				ctx = nproto.NewIncomingContextWithMD(ctx, payload.MD)
			}

			// Handle.
			if err := sub.handler(ctx, payload.MsgData); err != nil {
				// NOTE: do not print handle's error log. Let the handler do it.
				// dc.logger.Error().Err(err).Msg("MsgHandler error")
				return
			}

			// Ack only if no error.
			m.Ack()
			return

		}); err != nil {
			dc.logger.Error().Err(err).Msg("Submit handler failed")
		}
	}

	// Collect options.
	opts := []stan.SubscriptionOption{}
	opts = append(opts, sub.stanOptions...)
	opts = append(opts, stan.SetManualAckMode())     // Use manual ack mode. See above handler.
	opts = append(opts, stan.DurableName(sub.queue)) // Queue as durable name.

	// Sub logger.
	logger := dc.logger.With().
		Str("message_bus.destination", sub.subject).
		Str("message_bus.queue", sub.queue).
		Logger()

	// Subscription task.
	var task func()
	task = func() {
		// We don't need the returned stan.Subscription object since no Unsubscribe.
		_, err := sc.QueueSubscribe(fullSubject, sub.queue, handler, opts...)
		if err == nil {
			logger.Info().Msg("Subscription success")
			sub.subscribeCb(sc, sub.subject, sub.queue)
			return
		}

		logger.Error().Err(err).Msg("Subscription error")
		select {
		case <-stalec:
			logger.Info().Msg("Subscription stale")
			return
		case <-time.After(sub.retryWait):
			// Submit this subscription task again after some while.
			if err := dc.sRunner.Submit(task); err != nil {
				// NOTE: sRunner is never closed, so there should be no error.
				panic(err)
			}
		}
	}

	// Submit.
	if err := dc.sRunner.Submit(task); err != nil {
		// NOTE: sRunner is never closed, so there should be no error.
		panic(err)
	}

}

// reset gets lock then reset stan connection.
// It returns the old stan connection to be released (if not nil) and current subscriptions.
// It returns an error only when closed.
func (dc *DurConn) reset(sc stan.Conn, stalec chan struct{}) (
	oldSc stan.Conn,
	oldStalec chan struct{},
	subs []*subscription,
	err error) {

	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.closed {
		return nil, nil, nil, ErrClosed
	}

	oldSc = dc.sc
	oldStalec = dc.stalec
	dc.sc = sc
	dc.stalec = stalec

	// NOTE: Since subs is append-only, we can simply get the full slice as a snapshot for resubscribing.
	subs = dc.subs[:]
	return
}

// addSub gets lock then adds subscription to DurConn.
// It returns current stan connection.
func (dc *DurConn) addSub(sub *subscription) (sc stan.Conn, stalec chan struct{}, err error) {
	key := [2]string{sub.subject, sub.queue}
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.closed {
		return nil, nil, ErrClosed
	}

	if _, found := dc.subNames[key]; found {
		return nil, nil, ErrDupSubscription(sub.subject, sub.queue)
	}

	dc.subs = append(dc.subs, sub)
	dc.subNames[key] = len(dc.subs) - 1
	sc = dc.sc
	stalec = dc.stalec
	return
}

// close_ gets lock then sets closed to true.
// It returns the old stan connection to be released (if not nil).
func (dc *DurConn) close_() (oldSc stan.Conn, oldStalec chan struct{}, err error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.closed {
		return nil, nil, ErrClosed
	}

	oldSc = dc.sc
	oldStalec = dc.stalec
	dc.sc = nil
	dc.stalec = nil

	dc.closed = true
	return
}
