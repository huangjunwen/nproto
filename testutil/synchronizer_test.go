package testutil

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestForceOrder(t *testing.T) {

	assert := assert.New(t)
	c := NewTstSynchronizerController()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	pingCount := 0
	go func() {
		syncer := c.NewTstSynchronizer()
		for i := 0; i < 6; i++ {
			syncer.Yield("ping", nil)
			pingCount++
			time.Sleep(100 * time.Millisecond) // ping should be much slower in normal case since this sleep
		}
		wg.Done()
	}()

	pongCount := 0
	go func() {
		syncer := c.NewTstSynchronizer()
		for i := 0; i < 6; i++ {
			syncer.Yield("pong", nil)
			pongCount++
		}
		wg.Done()
	}()

	c.Expect("ping").Do(func(y *Yield) { assert.Equal(0, pingCount) }).Resume(nil)
	c.Expect("ping").Do(func(y *Yield) { assert.Equal(1, pingCount) }).Resume(nil)
	c.Expect("ping").Do(func(y *Yield) { assert.Equal(2, pingCount) }).Resume(nil)

	c.Expect("pong").Do(func(y *Yield) { assert.Equal(0, pongCount) }).Resume(nil)
	c.Expect("pong").Do(func(y *Yield) { assert.Equal(1, pongCount) }).Resume(nil)
	c.Expect("pong").Do(func(y *Yield) { assert.Equal(2, pongCount) }).Resume(nil)

	c.Expect("ping").Do(func(y *Yield) { assert.Equal(3, pingCount) }).Resume(nil)
	c.Expect("pong").Do(func(y *Yield) { assert.Equal(3, pongCount) }).Resume(nil)
	c.Expect("ping").Do(func(y *Yield) { assert.Equal(4, pingCount) }).Resume(nil)
	c.Expect("pong").Do(func(y *Yield) { assert.Equal(4, pongCount) }).Resume(nil)
	c.Expect("ping").Do(func(y *Yield) { assert.Equal(5, pingCount) }).Resume(nil)
	c.Expect("pong").Do(func(y *Yield) { assert.Equal(5, pongCount) }).Resume(nil)

	wg.Wait()

}

func TestPayload(t *testing.T) {

	assert := assert.New(t)
	c := NewTstSynchronizerController()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		syncer := c.NewTstSynchronizer()
		x := syncer.Yield("echo", nil)
		syncer.Yield("echo", x)
		wg.Done()
	}()

	c.Expect("echo").Do(func(y *Yield) { assert.Equal(nil, y.Payload()) }).Resume(1024)
	c.Expect("echo").Do(func(y *Yield) { assert.Equal(1024, y.Payload()) }).Resume(nil)
	wg.Wait()

}

func BenchmarkNoop(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewNoopSynchronizer().Yield("hello", "world")
	}
}
