package testutil

import (
	"errors"

	"github.com/nats-io/nuid"
)

var (
	ErrResume = errors.New("TstSynchronizer: Resume already resumed yield")
)

// Synchronizer is used to sync between go routines.
type Synchronizer interface {
	// Yield pause current execution and wait for continue. Optionally with some data attached.
	Yield(label string, payload interface{}) interface{}
}

/******************************************
	NoopSynchronizer is used for production
*******************************************/

// NoopSynchronizer do nothing. Used for production.
type NoopSynchronizer struct{}

var noopSynchronizer = NoopSynchronizer{}

// NewNoopSynchronizer returns NoopSynchronizer.
func NewNoopSynchronizer() Synchronizer {
	return noopSynchronizer
}

// Yield do nothing and continue immediately.
func (s NoopSynchronizer) Yield(_ string, _ interface{}) interface{} { return nil }

/******************************************
	TstSynchronizer is used for testing
*******************************************/

// TstSynchronizerController is used for sync TstSynchronizer. Used for testing.
type TstSynchronizerController struct {
	yieldC  chan *Yield
	pending map[string][]*Yield // label -> list of pending yields
}

// TstSynchronizer is used for testing.
type TstSynchronizer struct {
	id      string
	yieldC  chan *Yield
	resumeC chan interface{}
}

// Yield is raised from TstSynchronizer.Yield.
type Yield struct {
	syncer  *TstSynchronizer
	label   string
	payload interface{}
	resumed bool
}

// NewTstSynchronizerController returns a new TstSynchronizerController.
func NewTstSynchronizerController() *TstSynchronizerController {
	return &TstSynchronizerController{
		yieldC:  make(chan *Yield),
		pending: make(map[string][]*Yield),
	}
}

// NewTstSynchronizer returns a new TstSynchronizer.
func (c *TstSynchronizerController) NewTstSynchronizer() Synchronizer {
	return &TstSynchronizer{
		id:      nuid.Next(),
		yieldC:  c.yieldC,
		resumeC: make(chan interface{}),
	}
}

// Expect waits the expected labeled yield.
func (c *TstSynchronizerController) Expect(label string) *Yield {
	// First check pending ones.
	yields := c.pending[label]
	if len(yields) > 0 {
		// Pop the last
		yield := yields[len(yields)-1]
		if len(yields) == 1 {
			delete(c.pending, label)
		} else {
			c.pending[label] = yields[:len(yields)-1]
		}
		return yield
	}

	// Wait until the expected labeled yield is received.
	for {
		yield := <-c.yieldC
		if yield.label == label {
			return yield
		}

		// Add to pendings.
		c.pending[yield.label] = append(c.pending[yield.label], yield)
	}

}

// Yield implements Synchronizer interface.
func (s *TstSynchronizer) Yield(label string, payload interface{}) interface{} {
	s.yieldC <- &Yield{
		syncer:  s,
		label:   label,
		payload: payload,
	}
	return <-s.resumeC
}

// Resume the execution of the yield.
func (y *Yield) Resume(v interface{}) {
	if y.resumed {
		panic(ErrResume)
	}
	y.resumed = true
	y.syncer.resumeC <- v
}

// Label returns the label of the yield.
func (y *Yield) Label() string {
	return y.label
}

// Payload returns the payload of the yield.
func (y *Yield) Payload() interface{} {
	return y.payload
}

// ID returns the id of the Synchronizer.
func (y *Yield) ID() string {
	return y.syncer.id
}

// Resumed returns true if the yield has been resumed.
func (y *Yield) Resumed() bool {
	return y.resumed
}

// Do runs a function on y and returns y.
func (y *Yield) Do(fn func(*Yield)) *Yield {
	fn(y)
	return y
}
