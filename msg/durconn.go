package libmsg

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nuid"
	"github.com/rs/zerolog"
)

var (
	DefaultDurConnReconnectWait  = 5 * time.Second
	DefaultDurConnSubsResubsWait = 5 * time.Second
)

var (
	// ErrNotConnected is returned when the underly stan.Conn is not ready.
	ErrNotConnected = errors.New("nproto.libmsg.DurConn: not yet connected to streaming server")
	// ErrEmptyGroupName is returned when an empty group name is provided in subscription.
	ErrEmptyGroupName = errors.New("nproto.libmsg.DurConn: empty group name")
)

// DurConn provides re-connection/re-subscription functions on top of stan.Conn.
// It supports Publish/PublishAsync and durable QueueSubscribe (not support unsubscribing).
type DurConn struct {
	// Options.
	stanOptions   []stan.Option
	reconnectWait time.Duration
	logger        zerolog.Logger

	// Immutable fields.
	id        string
	nc        *nats.Conn
	clusterID string

	// At most one connect/Close can be run at any time.
	connectMu sync.Mutex

	// Mutable fields
	mu       sync.RWMutex
	sc       stan.Conn                   // nil if not connected/reconnecting
	scStaleC chan struct{}               // scStaleC is closed when sc is stale
	subs     map[[2]string]*subscription // (subject, group)->subscription
	closed   bool
}

// subscription is a single subscription. NOTE: Not support unsubscribing.
type subscription struct {
	// Options.
	stanOptions []stan.SubscriptionOption
	resubsWait  time.Duration

	// Immutable fields.
	conn    *DurConn
	subject string
	group   string
	cb      stan.MsgHandler
}

// DurConnOption is the option in creating DurConn.
type DurConnOption func(*DurConn)

// DurConnSubsOption is the option in DurConn.QueueSubscribe.
type DurConnSubsOption func(*subscription)

var (
	_ MsgSink = (*DurConn)(nil)
)

// NewDurConn creates a new DurConn. `nc` should have MaxReconnects < 0 set (e.g. Always reconnect).
func NewDurConn(nc *nats.Conn, clusterID string, opts ...DurConnOption) *DurConn {

	c := &DurConn{
		reconnectWait: DefaultDurConnReconnectWait,
		logger:        zerolog.Nop(),
		id:            nuid.Next(), // UUID as client id
		nc:            nc,
		clusterID:     clusterID,
		subs:          make(map[[2]string]*subscription),
	}
	for _, opt := range opts {
		opt(c)
	}

	// Start to connect.
	c.connect(false)
	return c
}

// connect is used to close old connection (if any), release old resouces then
// reconnect and resubscribe.
func (c *DurConn) connect(wait bool) {
	// Start a new routine to run.
	cfh.Go("DurConn.connect", func() {
		logger := c.logger.With().Str("fn", "connect").Logger()

		// Wait a while.
		if wait {
			time.Sleep(c.reconnectWait)
		}

		// At most one connect is allowed to run at any time.
		c.connectMu.Lock()
		defer c.connectMu.Unlock()

		// Reset and release old resources.
		cfh.Suspend("DurConn.connect:before.reset", c)
		c.mu.Lock()

		// DurConn is closed. Abort.
		if c.closed {
			c.mu.Unlock()
			cfh.Suspend("DurConn.connect:closed", c)
			logger.Info().Msg("Closed. Abort connection.")
			return
		}

		sc := c.sc
		scStaleC := c.scStaleC
		c.sc = nil
		c.scStaleC = nil

		c.mu.Unlock()
		cfh.Suspend("DurConn.connect:after.reset", c)

		if sc != nil {
			sc.Close()
		}

		if scStaleC != nil {
			// Notify stale of sc.
			close(scStaleC)
		}

		// Start to connect.
		opts := []stan.Option{}
		opts = append(opts, c.stanOptions...)
		opts = append(opts, stan.NatsConn(c.nc))
		opts = append(opts, stan.SetConnectionLostHandler(func(_ stan.Conn, _ error) {
			// Reconnect when connection lost.
			c.connect(true)
		}))

		cfh.Suspend("DurConn.connect:before.connect", c)
		sc, err := stanConnect(c.clusterID, c.id, opts...)
		if err != nil {
			// Reconnect when connection failed.
			cfh.Suspend("DurConn.connect:connect.failed", c)
			logger.Error().Err(err).Msg("Fail to connect.")
			c.connect(true)
			return
		}
		logger.Info().Msg("Connected.")
		cfh.Suspend("DurConn.connect:connect.ok", c)

		// Connected. Update fields and start to re-subscribe.
		cfh.Suspend("DurConn.connect:before.update", c)
		c.mu.Lock()

		c.sc = sc
		c.scStaleC = make(chan struct{})
		for _, sub := range c.subs {
			c.queueSubscribe(sub, c.sc, c.scStaleC)
		}

		c.mu.Unlock()
		cfh.Suspend("DurConn.connect:after.update", c)
		return
	})

}

// Close DurConn.
func (c *DurConn) Close() {
	// Guarantee no connect is running.
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	// Set fields to blank. Set closed to true.
	cfh.Suspend("DurConn.Close:before.reset", c)
	c.mu.Lock()

	sc := c.sc
	scStaleC := c.scStaleC
	c.sc = nil
	c.scStaleC = nil
	c.closed = true

	c.mu.Unlock()
	cfh.Suspend("DurConn.Close:after.reset", c)

	if sc != nil {
		// Close old sc.
		// NOTE: The callback (SetConnectionLostHandler) will not be invoked on normal Conn.Close().
		sc.Close()
	}

	if scStaleC != nil {
		// Close scStaleC.
		close(scStaleC)
	}

	return
}

// Publish == stan.Conn.Publish
func (c *DurConn) Publish(subject string, data []byte) error {
	c.mu.RLock()
	sc := c.sc
	c.mu.RUnlock()
	if sc == nil {
		return ErrNotConnected
	}
	return sc.Publish(subject, data)
}

// PublishAsync == stan.Conn.PublishAsync
func (c *DurConn) PublishAsync(subject string, data []byte, ah stan.AckHandler) (string, error) {
	c.mu.RLock()
	sc := c.sc
	c.mu.RUnlock()
	if sc == nil {
		return "", ErrNotConnected
	}
	return sc.PublishAsync(subject, data, ah)
}

// QueueSubscribe creates a durable queue subscription. Will be re-subscription after reconnection.
// Can't unsubscribe. This function returns error only when parameter error.
func (c *DurConn) QueueSubscribe(subject, group string, cb stan.MsgHandler, opts ...DurConnSubsOption) error {
	if group == "" {
		return ErrEmptyGroupName
	}

	sub := &subscription{
		resubsWait: DefaultDurConnSubsResubsWait,
		conn:       c,
		subject:    subject,
		group:      group,
		cb:         cb,
	}
	for _, opt := range opts {
		opt(sub)
	}

	// Check duplication.
	key := [2]string{subject, group}

	cfh.Suspend("DurConn.QueueSubscribe:before.subscribe", c)
	c.mu.Lock()

	if c.subs[key] != nil {
		c.mu.Unlock()
		cfh.Suspend("DurConn.QueueSubscribe:duplicate.subscribe", c)
		return fmt.Errorf("nproto.libmsg.DurConn: subject=%+q group=%+q has already subscribed", subject, group)
	}
	c.subs[key] = sub
	if c.sc != nil {
		// If sc is not nil, start to subscribe here.
		c.queueSubscribe(sub, c.sc, c.scStaleC)
	}

	c.mu.Unlock()
	cfh.Suspend("DurConn.QueueSubscribe:after.subscribe", c)

	return nil

}

func (c *DurConn) queueSubscribe(sub *subscription, sc stan.Conn, scStaleC chan struct{}) {

	cfh.Go("DurConn.queueSubscribe", func() {
		logger := sub.conn.logger.With().Str("fn", "queueSubscribe").
			Str("subj", sub.subject).
			Str("grp", sub.group).Logger()

		// Group as DurableName
		opts := []stan.SubscriptionOption{}
		opts = append(opts, sub.stanOptions...)
		opts = append(opts, stan.DurableName(sub.group))

		// Keep re-subscribing util stale become true.
		stale := false
		for !stale {
			_, err := sc.QueueSubscribe(sub.subject, sub.group, sub.cb, opts...)
			if err == nil {
				cfh.Suspend("DurConn.queueSubscribe:ok", c)
				// Normal case.
				logger.Info().Msg("Subscribed.")
				return
			}

			// Error case.
			logger.Error().Err(err).Msg("Subscribe error.")

			// Wait a while.
			t := time.NewTimer(sub.resubsWait)
			select {
			case <-scStaleC:
				stale = true
			case <-t.C:
			}
			t.Stop()
		}
		cfh.Suspend("DurConn.queueSubscribe:stale", c)

	})

}

// Deliver implements MsgSink interface.
func (c *DurConn) Deliver(msgs []Msg, delivered []bool) {
	logger := c.logger.With().Str("fn", "Deliver").Logger()

	// Use a wait group for message delivery.
	wg := &sync.WaitGroup{}
	wg.Add(len(msgs))

	// Ack handler to record successful delivery.
	mu := &sync.Mutex{}
	successIds := []string{} // List of successful delivered ids.
	ackHandler := func(id string, err error) {
		if err == nil {
			mu.Lock()
			successIds = append(successIds, id)
			mu.Unlock()
		}
		wg.Done()
	}

	// Batch publish.
	id2i := map[string]int{} // id -> msg index
	errs := 0
	for i, msg := range msgs {
		id, err := c.PublishAsync(msg.Subject(), msg.Data(), ackHandler)
		if err != nil {
			errs++
			logger.Error().Err(err).Msg("")
		} else {
			id2i[id] = i
		}
	}

	// Subtract errs.
	if errs > 0 {
		wg.Add(-errs)
	}

	// Wait...
	wg.Wait()

	// Process result.
	for _, id := range successIds {
		delivered[id2i[id]] = true
	}

}

// DurConnOptConnectWait sets connection wait.
func DurConnOptConnectWait(t time.Duration) DurConnOption {
	return func(c *DurConn) {
		c.stanOptions = append(c.stanOptions, stan.ConnectWait(t))
	}
}

// DurConnOptPubAckWait sets publish ack time wait.
func DurConnOptPubAckWait(t time.Duration) DurConnOption {
	return func(c *DurConn) {
		c.stanOptions = append(c.stanOptions, stan.PubAckWait(t))
	}
}

// DurConnOptPings sets ping
func DurConnOptPings(interval, maxOut int) DurConnOption {
	return func(c *DurConn) {
		c.stanOptions = append(c.stanOptions, stan.Pings(interval, maxOut))
	}
}

// DurConnOptReconnectWait sets reconnection wait.
func DurConnOptReconnectWait(t time.Duration) DurConnOption {
	return func(c *DurConn) {
		c.reconnectWait = t
	}
}

// DurConnOptLogger sets logger.
func DurConnOptLogger(logger *zerolog.Logger) DurConnOption {
	return func(c *DurConn) {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		c.logger = logger.With().Str("comp", "nproto.libmsg.DurConn").Logger()
	}
}

// DurConnSubsOptMaxInflight sets max message that can be sent to subscriber before ack
func DurConnSubsOptMaxInflight(m int) DurConnSubsOption {
	return func(s *subscription) {
		s.stanOptions = append(s.stanOptions, stan.MaxInflight(m))
	}
}

// DurConnSubsOptAckWait sets server side ack wait.
func DurConnSubsOptAckWait(t time.Duration) DurConnSubsOption {
	return func(s *subscription) {
		s.stanOptions = append(s.stanOptions, stan.AckWait(t))
	}
}

// DurConnSubsOptManualAcks sets to manual ack mode.
func DurConnSubsOptManualAcks() DurConnSubsOption {
	return func(s *subscription) {
		s.stanOptions = append(s.stanOptions, stan.SetManualAckMode())
	}
}

// DurConnSubsOptResubscribeWait sets resubscriptions wait.
func DurConnSubsOptResubsWait(t time.Duration) DurConnSubsOption {
	return func(s *subscription) {
		s.resubsWait = t
	}
}
