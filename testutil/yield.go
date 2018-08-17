package testutil

import (
	"errors"
)

var (
	ErrResumed = errors.New("nproto.testutil: Resuming resumed yield")
)

// NoopYield do nothing. Used in production.
func NoopYield(label string, payload ...interface{}) []interface{} { return nil }

// YieldController is used for synchronize different go routines in testing.
type YieldController struct {
	yieldC  chan *Yield
	pending []*Yield
}

// Yield represents a single pause of execution.
type Yield struct {
	ctrl    *YieldController
	label   string
	payload []interface{}
	resumeC chan []interface{}
	resumed bool
}

// NewYieldController creates a new YieldController.
func NewYieldController() *YieldController {
	return &YieldController{
		yieldC:  make(chan *Yield),
		pending: make([]*Yield, 0),
	}
}

// Yield is used to pause current routine's execution. And optionally exchange some data.
// This method can be used concurrently.
func (ctrl *YieldController) Yield(label string, payload ...interface{}) []interface{} {
	y := &Yield{
		ctrl:    ctrl,
		label:   label,
		payload: payload,
		resumeC: make(chan []interface{}),
	}
	ctrl.yieldC <- y
	return <-y.resumeC
}

// ExpectFn waits an expected (tested by fn) yield.
// This method shouldn't be used concurrently.
func (ctrl *YieldController) ExpectFn(fn func(*Yield) bool) *Yield {
	// First check pending ones.
	for i, y := range ctrl.pending {
		if fn(y) {
			ctrl.pending = append(ctrl.pending[:i], ctrl.pending[i+1:]...)
			return y
		}
	}

	// Wati until the expected yield is received.
	for {
		y := <-ctrl.yieldC
		if fn(y) {
			return y
		}
		// Add to pending.
		ctrl.pending = append(ctrl.pending, y)
	}
}

// Expect waits an expected yield with given label.
// This method shouldn't be used concurrently.
func (ctrl *YieldController) Expect(label string) *Yield {
	return ctrl.ExpectFn(func(y *Yield) bool {
		return y.label == label
	})
}

// Do something before resume.
// This method shouldn't be used concurrently.
func (y *Yield) Do(fn func(*Yield)) *Yield {
	fn(y)
	return y
}

// Resume the yield.
// This method shouldn't be used concurrently.
func (y *Yield) Resume(res ...interface{}) {
	if y.resumed {
		panic(ErrResumed)
	}
	y.resumed = true
	y.resumeC <- res
}

// Label of the yield.
func (y *Yield) Label() string {
	return y.label
}

// Payload of the yield.
func (y *Yield) Payload() []interface{} {
	return y.payload
}

// Resumed returns true if the yield has been resumed already.
func (y *Yield) Resumed() bool {
	return y.resumed
}
