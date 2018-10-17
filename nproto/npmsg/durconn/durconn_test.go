package durconn

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

// ----------- Mock begins -----------

type MockConn struct {
	calls []mockCall
}

type mockCall struct {
	Method string
	Args   []interface{}
}

var (
	_ stan.Conn = (*MockConn)(nil)
)

var (
	evilErr   = errors.New("Evil error")
	evilData  = []byte("evil")
	evil2Data = []byte("evil2")
)

func randWait() {
	// Wait 1ms~10ms
	time.Sleep(time.Duration(rand.Intn(10)+1) * time.Millisecond)
}

func newMockConn() *MockConn {
	return &MockConn{}
}

func (mc *MockConn) Publish(subject string, data []byte) error {
	mc.calls = append(mc.calls, mockCall{
		Method: "Publish",
		Args:   []interface{}{subject, data},
	})
	randWait()
	if bytes.Equal(data, evilData) {
		return evilErr
	}
	return nil
}

func (mc *MockConn) PublishAsync(subject string, data []byte, ah stan.AckHandler) (string, error) {
	mc.calls = append(mc.calls, mockCall{
		Method: "PublishAsync",
		Args:   []interface{}{subject, data, ah},
	})
	if bytes.Equal(data, evilData) {
		return "", evilErr
	}
	id := xid.New().String()
	go func() {
		randWait()
		if bytes.Equal(data, evil2Data) {
			ah(id, evilErr)
		} else {
			ah(id, nil)
		}
	}()
	return id, nil
}

func (mc *MockConn) Subscribe(subject string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	panic("Not implemented")
}

func (mc *MockConn) QueueSubscribe(subject, queue string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	mc.calls = append(mc.calls, mockCall{
		Method: "QueueSubscribe",
		Args:   []interface{}{subject, queue, cb, opts},
	})
	randWait()
	if queue == string(evilData) {
		return nil, evilErr
	} else if queue == string(evil2Data) {
		if len(mc.calls) > 3 {
			return nil, nil
		}
		return nil, evilErr
	} else {
		return nil, nil
	}
}

func (mc *MockConn) Close() error {
	mc.calls = append(mc.calls, mockCall{
		Method: "Close",
		Args:   []interface{}{},
	})
	return nil
}

func (mc *MockConn) NatsConn() *nats.Conn {
	panic("Not implemented")
}

// ----------- Mock ends -----------

// Test reset/subscribe/close_ .
// They are the 'atomic' methods of DurConn since they all use:
//
//   dc.mu.Lock()
//   defer dc.mu.Unlock()
//
// for the whole function.
func TestLockMethods(t *testing.T) {
	assert := assert.New(t)

	dc := &DurConn{
		subNames: map[[2]string]int{},
	}
	sub1 := &subscription{
		subject: "subject",
		queue:   "good1",
	}
	sub2 := &subscription{
		subject: "subject",
		queue:   "good2",
	}
	sub3 := &subscription{
		subject: "subject",
		queue:   "good3",
	}
	sc1 := newMockConn()
	sc2 := newMockConn()
	expect := func(closed bool, sc stan.Conn, subsLen int) {
		if closed {
			assert.True(dc.closed)
		} else {
			assert.False(dc.closed)
		}
		if sc != nil {
			assert.Equal(sc, dc.sc)
			assert.NotNil(dc.scStaleC)
		} else {
			assert.Nil(dc.sc)
			assert.Nil(dc.scStaleC)
		}
		assert.Len(dc.subs, subsLen)
		assert.Len(dc.subNames, subsLen)
	}

	// Init state.
	{
		expect(false, nil, 0)
	}

	// Reset to nil. Nothing changed.
	{
		sc, scStaleC, subs, err := dc.reset(nil, nil)
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Len(subs, 0)
		assert.Nil(err)
		expect(false, nil, 0)
	}

	// Add subscription 1 without stan connection.
	{
		sc, scStaleC, err := dc.subscribe(sub1)
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Nil(err)
		expect(false, nil, 1)
	}

	// Reset to stan connection 1.
	{
		sc, scStaleC, subs, err := dc.reset(sc1, make(chan struct{}))
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Len(subs, 1)
		assert.Nil(err)
		expect(false, sc1, 1)
	}

	// Add subscription 2 with stan connection.
	{
		sc, scStaleC, err := dc.subscribe(sub2)
		assert.Equal(sc1, sc)
		assert.NotNil(scStaleC)
		assert.Nil(err)
		expect(false, sc1, 2)
	}

	// Add subscription 1 again should result an error.
	{
		sc, scStaleC, err := dc.subscribe(sub1)
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.NotNil(err)
		expect(false, sc2, 2)
	}

	// Reset to stan connection 2.
	{
		sc, scStaleC, subs, err := dc.reset(sc2, make(chan struct{}))
		assert.Equal(sc1, sc)
		assert.NotNil(scStaleC)
		assert.Len(subs, 2)
		assert.Nil(err)
		expect(false, sc2, 2)
	}

	// Now close.
	{
		sc, scStaleC, err := dc.close_()
		assert.Equal(sc2, sc)
		assert.NotNil(scStaleC)
		assert.Nil(err)
		expect(true, nil, 2)
	}

	// Further calls should result errors.
	{
		sc, scStaleC, subs, err := dc.reset(sc1, make(chan struct{}))
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Nil(subs)
		assert.Error(err)
	}
	{
		sc, scStaleC, err := dc.subscribe(sub3)
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Error(err)
	}
	{
		sc, scStaleC, err := dc.close_()
		assert.Nil(sc)
		assert.Nil(scStaleC)
		assert.Error(err)
	}
	expect(true, nil, 2)
}
