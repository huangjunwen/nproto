package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestForceOrder(t *testing.T) {

	assert := assert.New(t)
	cfh := NewTestControlFlowHook()

	pingCount := 0
	cfh.Go("PING", func() {
		func() {
			// Nest function should be the same.
			for i := 0; i < 3; i++ {
				cfh.Suspend("ping", nil)
				pingCount++
				time.Sleep(100 * time.Millisecond) // ping should be much slower in normal case since this sleep
			}
		}()
		for i := 0; i < 3; i++ {
			cfh.Suspend("ping", nil)
			pingCount++
			time.Sleep(100 * time.Millisecond) // ping should be much slower in normal case since this sleep
		}
	})

	pongCount := 0
	cfh.Go("PONG", func() {
		for i := 0; i < 6; i++ {
			cfh.Suspend("pong", nil)
			pongCount++
		}
	})

	pong := cfh.Capture("PONG")
	ping := cfh.Capture("PING")

	ping.Expect("ping").Do(func(_ interface{}) { assert.Equal(0, pingCount) }).
		Expect("ping").Do(func(_ interface{}) { assert.Equal(1, pingCount) }).
		Expect("ping").Do(func(_ interface{}) { assert.Equal(2, pingCount) })

	pong.Expect("pong").Do(func(_ interface{}) { assert.Equal(0, pongCount) }).
		Expect("pong").Do(func(_ interface{}) { assert.Equal(1, pongCount) }).
		Expect("pong").Do(func(_ interface{}) { assert.Equal(2, pongCount) })

	ping.Expect("ping").Do(func(_ interface{}) { assert.Equal(3, pingCount) })
	pong.Expect("pong").Do(func(_ interface{}) { assert.Equal(3, pongCount) })
	ping.Expect("ping").Do(func(_ interface{}) { assert.Equal(4, pingCount) })
	pong.Expect("pong").Do(func(_ interface{}) { assert.Equal(4, pongCount) })
	ping.Expect("ping").Do(func(_ interface{}) { assert.Equal(5, pingCount) }).Resume(nil)
	pong.Expect("pong").Do(func(_ interface{}) { assert.Equal(5, pongCount) }).Resume(nil)

	cfh.Wait()

}
