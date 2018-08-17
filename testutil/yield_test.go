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
			ctrl.Yield("ping", nil)
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

	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(0, pingCount) }).Resume(nil)
	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(1, pingCount) }).Resume(nil)
	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(2, pingCount) }).Resume(nil)

	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(0, pongCount) }).Resume(nil)
	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(1, pongCount) }).Resume(nil)
	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(2, pongCount) }).Resume(nil)

	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(3, pingCount) }).Resume(nil)
	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(3, pongCount) }).Resume(nil)
	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(4, pingCount) }).Resume(nil)
	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(4, pongCount) }).Resume(nil)
	ctrl.Expect("ping").Do(func(y *Y) { assert.Equal(5, pingCount) }).Resume(nil)
	ctrl.Expect("pong").Do(func(y *Y) { assert.Equal(5, pongCount) }).Resume(nil)

	wg.Wait()
}

func TestPayload(t *testing.T) {

	assert := assert.New(t)
	ctrl := NewYieldController()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		x := ctrl.Yield("echo", nil)
		ctrl.Yield("echo", x)
		wg.Done()
	}()

	ctrl.Expect("echo").Do(func(y *Y) { assert.Nil(y.Payload()) }).Resume(1024)
	ctrl.Expect("echo").Do(func(y *Y) { assert.Equal(1024, y.Payload()) }).Resume(nil)
	wg.Wait()

}

func BenchmarkNoop(b *testing.B) {
	yield := NoopYield
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		yield("hello", "world")
	}
}
