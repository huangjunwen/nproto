package durconn

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
		cfh = oldCfh
		stanConnect = oldStanConnect
	}()

	// Create DurConn.
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

	// ### Subscribe: emulate normal subscription.
	{
		tcfh.Go("Subscribe#1", func() {
			c.Subscribe("subj", "g1", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})

		tcfh.Capture("Subscribe#1").
			Expect("DurConn.Subscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Empty(c.subs)
				assert.False(c.closed)
			}).
			Expect("DurConn.Subscribe:after.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Len(c.subs, 1)
				assert.False(c.closed)
			}).Resume(nil)

		tcfh.Capture("DurConn.subscribe").
			Expect("MockConn.QueueSubscribe").
			Resume(nil).
			Expect("DurConn.subscribe:ok").
			Resume(nil)

		tcfh.Wait()
	}

	// ### Subscribe: emulate duplicate subscription.
	{
		tcfh.Go("Subscribe#2", func() {
			c.Subscribe("subj", "g1", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})

		tcfh.Capture("Subscribe#2").
			Expect("DurConn.Subscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
				assert.Len(c.subs, 1)
				assert.False(c.closed)
			}).
			Expect("DurConn.Subscribe:duplicate.subscribe").
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
		tcfh.Go("Subscribe#3", func() {
			c.Subscribe("subj", "g3", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})
		tcfh.Go("Subscribe#4", func() {
			c.Subscribe("subj", "g4", nil, DurConnSubsOptResubsWait(100*time.Millisecond))
		})
		c.sc.(*MockConn).Options.ConnectionLostCB(c.sc, errors.New("Mock conn lost")) // Emulate connection lost.

		// The first half of subscription 3.
		tcfh.Capture("Subscribe#3").
			Expect("DurConn.Subscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 1)
				assert.NotNil(c.sc)
				assert.NotNil(c.scStaleC)
			}).
			Expect("DurConn.Subscribe:after.subscribe").
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
		tcfh.Capture("DurConn.subscribe").
			Expect("MockConn.QueueSubscribe").
			Resume(errors.New("MockConn stale")).
			Expect("DurConn.subscribe:stale").
			Resume(nil)

		// Subscription 4 starts now.
		tcfh.Capture("Subscribe#4").
			Expect("DurConn.Subscribe:before.subscribe").
			Do(func(_ interface{}) {
				assert.Len(c.subs, 2)
				assert.Nil(c.sc)
				assert.Nil(c.scStaleC)
			}).
			Expect("DurConn.Subscribe:after.subscribe").
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
		tcfh.Capture("DurConn.subscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.subscribe:ok").
			Resume(nil)

		tcfh.Capture("DurConn.subscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.subscribe:ok").
			Resume(nil)

		tcfh.Capture("DurConn.subscribe").
			Expect("MockConn.QueueSubscribe").
			Expect("DurConn.subscribe:ok").
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
