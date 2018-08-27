package libmsg

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nuid"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/testutil"
	"github.com/huangjunwen/nproto/util"
)

var (
	ErrNotImplemented = errors.New("Not implemented")
)

// MockConn mocks stan.Conn.
type MockConn struct {
	Options stan.Options
	cfh     util.ControlFlowHook
}

// MockSubscription mocks stan.subscription.
type MockSubscription struct{}

var (
	_ stan.Conn         = (*MockConn)(nil)
	_ stan.Subscription = (*MockSubscription)(nil)
)

func MakeMockConnect(cfh util.ControlFlowHook) func(string, string, ...stan.Option) (stan.Conn, error) {
	return func(stanClusterID, clientID string, options ...stan.Option) (stan.Conn, error) {
		c := &MockConn{
			cfh: cfh,
		}
		for _, option := range options {
			if err := option(&c.Options); err != nil {
				return nil, err
			}
		}
		err, ok := cfh.Suspend("MockConnect", c).(error)
		if !ok {
			return c, nil
		}
		return nil, err
	}
}

func (c *MockConn) Publish(subject string, data []byte) error {
	err, ok := c.cfh.Suspend("MockConn.Publish", []interface{}{c, subject, data}).(error)
	if !ok {
		return nil
	}
	return err
}

func (c *MockConn) PublishAsync(subject string, data []byte, ah stan.AckHandler) (string, error) {
	err, ok := c.cfh.Suspend("MockConn.PublishAsync", []interface{}{c, subject, data, ah}).(error)
	if !ok {
		return nuid.Next(), nil
	}
	return "", err
}

func (c *MockConn) Subscribe(subject string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	err, ok := c.cfh.Suspend("MockConn.Subscribe", []interface{}{c, subject, cb, opts}).(error)
	if !ok {
		return &MockSubscription{}, nil
	}
	return nil, err
}

func (c *MockConn) QueueSubscribe(subject, qgroup string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	err, ok := c.cfh.Suspend("MockConn.QueueSubscribe", []interface{}{c, subject, qgroup, cb, opts}).(error)
	if !ok {
		return &MockSubscription{}, nil
	}
	return nil, err
}

func (c *MockConn) Close() error {
	err, ok := c.cfh.Suspend("MockConn.Close", c).(error)
	if !ok {
		return nil
	}
	return err
}

func (c *MockConn) NatsConn() *nats.Conn {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) ClearMaxPending() error {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) Delivered() (int64, error) {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) Dropped() (int, error) {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) IsValid() bool {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) MaxPending() (int, int, error) {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) Pending() (int, int, error) {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) PendingLimits() (int, int, error) {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) SetPendingLimits(msgLimit, bytesLimit int) error {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) Unsubscribe() error {
	panic(ErrNotImplemented)
}

func (s *MockSubscription) Close() error {
	panic(ErrNotImplemented)
}

// ----------- Test -----------

func TestDurConn(t *testing.T) {

	assert := assert.New(t)
	tcfh := testutil.NewTestControlFlowHook()

	// Mock.
	oldCfh := cfh
	oldStanConnect := stanConnect
	cfh = tcfh
	stanConnect = MakeMockConnect(tcfh)
	defer func() {
		stanConnect = oldStanConnect
		cfh = oldCfh
	}()

	nc := &nats.Conn{}
	nc.Opts.MaxReconnect = -1
	c, err := NewDurConn(nc, "test", DurConnOptReconnectWait(100*time.Millisecond))
	assert.NoError(err)

	// ### Connect: emulate connection failure then success.
	{
		connTH := tcfh.Capture("DurConn.connect")
		connTH.
			Expect("DurConn.connect:before.reset").
			Do(func(_ interface{}) {
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
				assert.Empty(c.subs)
				assert.False(c.closed)
			}).
			Expect("DurConn.connect:after.reset").
			Expect("DurConn.connect:before.connect").
			Expect("MockConnect").
			Resume(errors.New("Mock connect failed")).
			Expect("DurConn.connect:connect.failed").
			Resume(nil)

		connTH = tcfh.Capture("DurConn.connect")
		connTH.
			Expect("DurConn.connect:before.reset").
			Do(func(_ interface{}) {
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
				assert.Empty(c.subs)
				assert.False(c.closed)
			}).
			Expect("DurConn.connect:after.reset").
			Expect("DurConn.connect:before.connect").
			Expect("MockConnect").
			Resume(nil).
			Expect("DurConn.connect:connect.ok").
			Expect("DurConn.connect:before.update").
			Expect("DurConn.connect:after.update").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Empty(c.subs)
				assert.False(c.closed)
			}).
			Resume(nil)

		tcfh.Wait()
	}

	// ### Subscribe: emulate normal subscribe.
	{
		tcfh.Go("QueueSubscribe#1", func() {
			c.QueueSubscribe("subj", "g1", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})

		tcfh.Capture("QueueSubscribe#1").
			Expect("DurConn.QueueSubscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Empty(c.subs)
				assert.False(c.closed)
			}).
			Expect("DurConn.QueueSubscribe:after.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Len(c.subs, 1)
				assert.False(c.closed)
			}).Resume(nil)

		tcfh.Capture("DurConn.queueSubscribe").
			Expect("MockConn.QueueSubscribe").
			Resume(nil).
			Expect("DurConn.queueSubscribe:ok").
			Resume(nil)

		tcfh.Wait()
	}

	// ### Subscribe: emulate duplicate subscribe.
	{
		tcfh.Go("QueueSubscribe#2", func() {
			c.QueueSubscribe("subj", "g1", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})

		tcfh.Capture("QueueSubscribe#2").
			Expect("DurConn.QueueSubscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Len(c.subs, 1)
				assert.False(c.closed)
			}).
			Expect("DurConn.QueueSubscribe:duplicate.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Len(c.subs, 1)
				assert.False(c.closed)
			}).
			Resume(nil)

		tcfh.Wait()
	}

	// ### Subscribe before/after connection is lost.
	{
		tcfh.Go("QueueSubscribe#3", func() {
			c.QueueSubscribe("subj", "g3", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})
		tcfh.Go("QueueSubscribe#4", func() {
			c.QueueSubscribe("subj", "g4", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})
		c.sc.(*MockConn).Options.ConnectionLostCB(c.sc, errors.New("Mock conn lost")) // Emulate connection lost.

		// The first half of subscription 3.
		tcfh.Capture("QueueSubscribe#3").
			Expect("DurConn.QueueSubscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 1)
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
			}).
			Expect("DurConn.QueueSubscribe:after.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 2)
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
			}).
			Resume(nil)

		// Now connection lost.
		connTH := tcfh.Capture("DurConn.connect")
		connTH.
			Expect("DurConn.connect:before.reset").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 2)
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
			}).
			Expect("DurConn.connect:after.reset").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 2)
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
			}).
			Expect("MockConn.Close").
			Expect("DurConn.connect:before.connect")

		// The second half of subscription 3 failed since stale.
		tcfh.Capture("DurConn.queueSubscribe").
			Expect("MockConn.QueueSubscribe").
			Resume(errors.New("MockConn stale")).
			Expect("DurConn.queueSubscribe:stale").
			Resume(nil)

		// Subscription 4 starts now.
		tcfh.Capture("QueueSubscribe#4").
			Expect("DurConn.QueueSubscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 2)
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
			}).
			Expect("DurConn.QueueSubscribe:after.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 3)
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
			}).
			Resume(nil)

		// Now reconnect.
		connTH.
			Expect("MockConnect").
			Expect("DurConn.connect:connect.ok").
			Expect("DurConn.connect:before.update").
			Expect("DurConn.connect:after.update").
			Resume(nil)

		// Re subscription 3 times.
		tcfh.Capture("DurConn.queueSubscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.queueSubscribe:ok").
			Resume(nil)

		tcfh.Capture("DurConn.queueSubscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.queueSubscribe:ok").
			Resume(nil)

		tcfh.Capture("DurConn.queueSubscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.queueSubscribe:ok").
			Resume(nil)

		tcfh.Wait()

	}

	// ### Close.
	{
		tcfh.Go("Close", c.Close)
		c.sc.(*MockConn).Options.ConnectionLostCB(c.sc, errors.New("Mock conn lost")) // Emulate reconnection.

		tcfh.Capture("Close").
			Expect("DurConn.Close:before.reset").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 3)
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.False(c.closed)
			}).
			Expect("DurConn.Close:after.reset").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 3)
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
				assert.True(c.closed)
			}).
			Expect("MockConn.Close").
			Resume(nil)

		tcfh.Capture("DurConn.connect").
			Expect("DurConn.connect:before.reset").
			Expect("DurConn.connect:closed").
			Resume(nil)

		tcfh.Wait()
	}

}
