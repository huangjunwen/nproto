package durconn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/nproto/msg"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nuid"
	"github.com/rs/zerolog"
)

var (
	DefaultSubjectPrefix         = "npmsg"
	DefaultDurConnReconnectWait  = 5 * time.Second
	DefaultDurConnSubsResubsWait = 5 * time.Second
)

var (
	ErrMaxReconnect      = errors.New("nproto.npmsg.durconn.DurConn: nc should have MaxReconnects < 0")
	ErrNotConnected      = errors.New("nproto.npmsg.durconn.DurConn: Not connect to streaming server")
	ErrDurConnSubsOption = errors.New("nproto.npmsg.durconn.DurConn: Expect DurConnSubsOption")
)

// DurConn provides re-connection/re-subscription functions on top of stan.Conn.
// It implements npmsg.MsgPublisher and npmsg.MsgSubscriber interfaces.
type DurConn struct {
	// Options.
	stanOptions   []stan.Option
	subjPrefix    string
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
	handler npmsg.MsgHandler
}

// DurConnOption is the option in creating DurConn.
type DurConnOption func(*DurConn) error

// DurConnSubsOption is the option in DurConn.QueueSubscribe.
type DurConnSubsOption func(*subscription) error

// stanMsg implements Msg interface.
type stanMsg stan.Msg

var (
	_ npmsg.MsgPublisher  = (*DurConn)(nil)
	_ npmsg.MsgSubscriber = (*DurConn)(nil)
)

// NewDurConn creates a new DurConn. `nc` should have MaxReconnects < 0 set (e.g. Always reconnect).
func NewDurConn(nc *nats.Conn, clusterID string, opts ...DurConnOption) (*DurConn, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrMaxReconnect
	}

	c := &DurConn{
		subjPrefix:    DefaultSubjectPrefix,
		reconnectWait: DefaultDurConnReconnectWait,
		logger:        zerolog.Nop(),
		id:            nuid.Next(), // UUID as client id
		nc:            nc,
		clusterID:     clusterID,
		subs:          make(map[[2]string]*subscription),
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// Start to connect.
	c.connect(false)
	return c, nil
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
			c.subscribe(sub, c.sc, c.scStaleC)
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

// Publish implements MsgPublisher interface.
func (c *DurConn) Publish(msg npmsg.Msg) error {
	c.mu.RLock()
	sc := c.sc
	c.mu.RUnlock()
	if sc == nil {
		return ErrNotConnected
	}
	return sc.Publish(c.makeSubject(msg.Subject()), msg.Data())
}

// PublishBatch implements MsgPublisher interface.
func (c *DurConn) PublishBatch(msgs []npmsg.Msg) (errors []error) {

	errors = make([]error, len(msgs))

	// Check sc.
	c.mu.RLock()
	sc := c.sc
	c.mu.RUnlock()
	if sc == nil {
		for i, _ := range msgs {
			errors[i] = ErrNotConnected
		}
		return
	}

	// Use a wait group to wait message deliveries.
	wg := &sync.WaitGroup{}
	wg.Add(len(msgs))

	// Ack handler to record result.
	mu := &sync.Mutex{}
	ackResult := map[string]error{} // id -> error
	ackHandler := func(id string, err error) {
		mu.Lock()
		ackResult[id] = err
		mu.Unlock()
		wg.Done()
	}

	// Publish async.
	id2i := map[string]int{} // id -> msg index
	nErrs := 0
	for i, msg := range msgs {
		id, err := sc.PublishAsync(c.makeSubject(msg.Subject()), msg.Data(), ackHandler)
		if err != nil {
			nErrs++
			errors[i] = err
		} else {
			id2i[id] = i
		}
	}

	// Subtract nErrs and wait.
	if nErrs > 0 {
		wg.Add(-nErrs)
	}
	wg.Wait()

	// Process ackResult.
	for id, err := range ackResult {
		errors[id2i[id]] = err
	}
	return
}

func (c *DurConn) Subscribe(subject, group string, handler npmsg.MsgHandler, opts ...interface{}) error {

	for _, opt := range opts {
		if _, ok := opt.(DurConnSubsOption); !ok {
			return ErrDurConnSubsOption
		}
	}

	sub := &subscription{
		resubsWait: DefaultDurConnSubsResubsWait,
		conn:       c,
		subject:    subject,
		group:      group,
		handler:    handler,
	}
	for _, opt := range opts {
		if err := opt.(DurConnSubsOption)(sub); err != nil {
			return err
		}
	}

	// Check duplication.
	key := [2]string{subject, group}

	cfh.Suspend("DurConn.Subscribe:before.subscribe", c)
	c.mu.Lock()

	if c.subs[key] != nil {
		c.mu.Unlock()
		cfh.Suspend("DurConn.Subscribe:duplicate.subscribe", c)
		return fmt.Errorf("nproto.npmsg.durconn.DurConn: Duplicated subscription subject=%+q group=%+q", subject, group)
	}
	c.subs[key] = sub
	if c.sc != nil {
		// If sc is not nil, start to subscribe here.
		c.subscribe(sub, c.sc, c.scStaleC)
	}

	c.mu.Unlock()
	cfh.Suspend("DurConn.Subscribe:after.subscribe", c)

	return nil

}

func (c *DurConn) subscribe(sub *subscription, sc stan.Conn, scStaleC chan struct{}) {

	handler := func(msg *stan.Msg) {
		if err := sub.handler(context.Background(), (*stanMsg)(msg)); err != nil {
			sub.conn.logger.Error().Str("fn", "subscribe.MsgHandler").
				Str("subj", sub.subject).
				Str("grp", sub.group).Err(err).Msg("")
		} else {
			msg.Ack()
		}
	}

	cfh.Go("DurConn.subscribe", func() {
		logger := sub.conn.logger.With().Str("fn", "subscribe").
			Str("subj", sub.subject).
			Str("grp", sub.group).Logger()

		// Group as DurableName
		opts := []stan.SubscriptionOption{}
		opts = append(opts, sub.stanOptions...)
		opts = append(opts, stan.SetManualAckMode())
		opts = append(opts, stan.DurableName(sub.group))

		// Keep re-subscribing util stale become true.
		stale := false
		for !stale {
			_, err := sc.QueueSubscribe(c.makeSubject(sub.subject), sub.group, handler, opts...)
			if err == nil {
				cfh.Suspend("DurConn.subscribe:ok", c)
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
		cfh.Suspend("DurConn.subscribe:stale", c)

	})

}

func (c *DurConn) makeSubject(subject string) string {
	return fmt.Sprintf("%s.%s", c.subjPrefix, subject)
}

// DurConnOptConnectWait sets connection wait.
func DurConnOptConnectWait(t time.Duration) DurConnOption {
	return func(c *DurConn) error {
		c.stanOptions = append(c.stanOptions, stan.ConnectWait(t))
		return nil
	}
}

// DurConnOptPubAckWait sets publish ack time wait.
func DurConnOptPubAckWait(t time.Duration) DurConnOption {
	return func(c *DurConn) error {
		c.stanOptions = append(c.stanOptions, stan.PubAckWait(t))
		return nil
	}
}

// DurConnOptPings sets ping
func DurConnOptPings(interval, maxOut int) DurConnOption {
	return func(c *DurConn) error {
		c.stanOptions = append(c.stanOptions, stan.Pings(interval, maxOut))
		return nil
	}
}

// DurConnOptSubjectPrefix sets the subject prefix.
func DurConnOptSubjectPrefix(subjPrefix string) DurConnOption {
	return func(c *DurConn) error {
		c.subjPrefix = subjPrefix
		return nil
	}
}

// DurConnOptReconnectWait sets reconnection wait.
func DurConnOptReconnectWait(t time.Duration) DurConnOption {
	return func(c *DurConn) error {
		c.reconnectWait = t
		return nil
	}
}

// DurConnOptLogger sets logger.
func DurConnOptLogger(logger *zerolog.Logger) DurConnOption {
	return func(c *DurConn) error {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		c.logger = logger.With().Str("comp", "nproto.npmsg.durconn.DurConn").Logger()
		return nil
	}
}

// DurConnSubsOptMaxInflight sets max message that can be sent to subscriber before ack
func DurConnSubsOptMaxInflight(m int) DurConnSubsOption {
	return func(s *subscription) error {
		s.stanOptions = append(s.stanOptions, stan.MaxInflight(m))
		return nil
	}
}

// DurConnSubsOptAckWait sets server side ack wait.
func DurConnSubsOptAckWait(t time.Duration) DurConnSubsOption {
	return func(s *subscription) error {
		s.stanOptions = append(s.stanOptions, stan.AckWait(t))
		return nil
	}
}

// DurConnSubsOptResubscribeWait sets resubscriptions wait.
func DurConnSubsOptResubsWait(t time.Duration) DurConnSubsOption {
	return func(s *subscription) error {
		s.resubsWait = t
		return nil
	}
}

// Subject implements Msg interface.
func (m *stanMsg) Subject() string {
	return (*stan.Msg)(m).Subject
}

// Data implements Msg interface.
func (m *stanMsg) Data() []byte {
	return (*stan.Msg)(m).Data
}
