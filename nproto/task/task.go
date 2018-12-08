package task

import (
	"errors"
	"sync"
)

var (
	ErrClosed  = errors.New("nproto.task.LimitedTaskRunner: Closed.")
	ErrTooBusy = errors.New("nproto.task.LimitedTaskRunner: Too busy.")
)

var (
	DefaultMaxConcurrency = 10 * 1024
)

// TaskRunner is an interface to run tasks.
type TaskRunner interface {
	// Run runs t. It should return an error if t can't be run.
	// The call MUST not block.
	Run(t func()) error
}

// LimitedTaskRunner limits max concurrency (go routines) to run tasks.
type LimitedTaskRunner struct {
	sem    chan struct{}
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

var (
	_ TaskRunner = (*LimitedTaskRunner)(nil)
)

// NewLimitedTaskRunner creates a new LimitedTaskRunner. If maxConcurrency <= 0,
// then DefaultMaxConcurrency will be used.
func NewLimitedTaskRunner(maxConcurrency int) *LimitedTaskRunner {
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	return &LimitedTaskRunner{
		sem: make(chan struct{}, maxConcurrency),
	}
}

func (r *LimitedTaskRunner) taskDone() {
	<-r.sem
	r.wg.Done()
}

// Run implements TaskRunner interface.
func (r *LimitedTaskRunner) Run(t func()) error {
	// Wait semaphore.
	select {
	case r.sem <- struct{}{}:
	default:
		return ErrTooBusy
	}

	// NOTE: Don't put this after checking closed. Otherwise
	//
	//   Time       thread 1 (Run)         thread 2 (Close)
	//    |            |                      |
	//    V            V                      V
	//            r.mu.RLock()           ...
	//            closed := r.closed     ...
	//            r.mu.RUnlock()         ...
	//            ...                    r.mu.Lock()
	//            ...                    r.closed = true
	//            ...                    r.mu.Unlock()
	//            ...                    r.wg.Wait()    <- at this moment wg is 0
	//            r.wg.Add(1)
	//            ...
	//            t()
	//            r.wg.Done()
	//
	// Then thread 2 (Close) will not wait `t()` to finish.
	r.wg.Add(1)

	// Check closed.
	r.mu.RLock()
	closed := r.closed
	r.mu.RUnlock()

	if closed {
		r.taskDone()
		return ErrClosed
	}

	go func() {
		defer r.taskDone()
		t()
	}()
	return nil

}

// Close stops the task runner and wait all running tasks to finish.
func (r *LimitedTaskRunner) Close() {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	r.wg.Wait()
}
