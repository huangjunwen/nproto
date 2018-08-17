package testutil

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestForceOrder(t *testing.T) {

	assert := assert.New(t)
	ctrl := NewYieldController()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	pingCount := 0
	go func() {
		for i := 0; i < 6; i++ {
			ctrl.Yield("ping")
			pingCount++
			time.Sleep(100 * time.Millisecond) // ping should be much slower in normal case since this sleep
		}
		wg.Done()
	}()

	pongCount := 0
	go func() {
		for i := 0; i < 6; i++ {
			ctrl.Yield("pong", nil)
			pongCount++
		}
		wg.Done()
	}()

	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(0, pingCount) }).Resume()
	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(1, pingCount) }).Resume()
	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(2, pingCount) }).Resume()

	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(0, pongCount) }).Resume()
	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(1, pongCount) }).Resume()
	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(2, pongCount) }).Resume()

	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(3, pingCount) }).Resume()
	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(3, pongCount) }).Resume()
	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(4, pingCount) }).Resume()
	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(4, pongCount) }).Resume()
	ctrl.Expect("ping").Do(func(y *Yield) { assert.Equal(5, pingCount) }).Resume()
	ctrl.Expect("pong").Do(func(y *Yield) { assert.Equal(5, pongCount) }).Resume()

	wg.Wait()
}

func TestPayload(t *testing.T) {

	assert := assert.New(t)
	ctrl := NewYieldController()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		x := ctrl.Yield("echo")
		ctrl.Yield("echo", x...)
		wg.Done()
	}()

	ctrl.Expect("echo").Do(func(y *Yield) { assert.Empty(y.Payload()) }).Resume(1024)
	ctrl.Expect("echo").Do(func(y *Yield) { assert.Equal([]interface{}{1024}, y.Payload()) }).Resume()
	wg.Wait()

}

func BenchmarkNoop(b *testing.B) {
	yield := NoopYield
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		yield("hello", "world")
	}
}
