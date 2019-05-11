package taskrunner

import (
	"errors"
	"sync"
)

var (
	// ErrClosed is returned when task is submitted after LimitedRunner has been closed.
	ErrClosed = errors.New("nproto.task.LimitedRunner: Closed.")
	// ErrTooBusy is returned when task is submitted but LimitedRunner is too busy to handle.
	ErrTooBusy = errors.New("nproto.task.LimitedRunner: Too busy.")
)

var (
	// DefaultMaxConcurrency is the default value of maxConcurrency.
	DefaultMaxConcurrency = 5 * 1024
)

// TaskRunner is an interface to run tasks.
type TaskRunner interface {
	// Submit submits a task to run. It must return an error if the task can't be run.
	// The call must not block.
	Submit(task func()) error

	// Close stops the TaskRunner and waits for all tasks finish.
	// Any Submit after Close should return an error.
	Close()
}

var (
	_ TaskRunner = (*LimitedRunner)(nil)
)

// LimitedRunner implements TaskRunner interface. It has two parameters:
//
//   - maxConcurrency: the maximum number of concurrent go routines, at least 1
//   - maxQueue: the maximum number of queued tasks, if < 0 then the queue is unlimited, 0 for no queue, > 0 for limited queue
//
// When task is submitted, if current number of go routines is less than maxConcurrency,
// then a new go routine will be started immediately to run the task. Otherwise if current
// number of queued tasks is less than maxQueue (or the queue is unlimited), then the task
// will be queued for later execution.
//
// Thus if there are some queued tasks, then current number of go routines must be maxConcurrency.
//
// When a go routine finish one task, it will check the queue, if there are queued tasks, the first
// task will be popped and run directly. It will exit only when there is no queued task.
type LimitedRunner struct {
	maxConcurrency int // maximum concurrent go routines: maxConcurrency > 0
	maxQueue       int // maximum queue tasks, unlimited if < 0

	cond   *sync.Cond
	mu     sync.Mutex // protect the following
	closed bool
	c      int       // current running go routines: c <= maxConcurrency
	q      taskQueue // current task queue: q.n <= maxQueue if maxQueue >= 0
}

// NOTE: Some implementation notes about LimitedRunner, there are three major mutex blocks:
//
//   - "add task" block: add task to run or queue.
//   - "done task" block: run queued task or quit.
//   - "close" block: set closed and wait. In fact, there are two mutex blocks here since
//                    cond.wait will release the lock internal.
//
// Correctness should be kept under any execution order of these blocks.
//
//   - After "close" block, any "add task" block will return ErrClosed, thus no new task will be added.
//     "done task" blocks will run all queued tasks then quit one by one until c become 0. The last
//     cond.Signal will make the condition become true.

type taskQueue struct {
	head *taskNode
	tail *taskNode
	n    int
}

type taskNode struct {
	task func()
	next *taskNode
}

// NewDefaultLimitedRunner creates a new LimitedRunner with DefaultMaxConcurrency and unlimited queued.
func NewDefaultLimitedRunner() *LimitedRunner {
	return NewLimitedRunner(DefaultMaxConcurrency, -1)
}

// NewLimitedRunner creates a new LimitedRunner. If maxConcurrency <= 0, then DefaultMaxConcurrency will be used.
func NewLimitedRunner(maxConcurrency, maxQueue int) *LimitedRunner {
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	r := &LimitedRunner{
		maxConcurrency: maxConcurrency,
		maxQueue:       maxQueue,
	}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// Submit implements TaskRunner interface. It returns error when queued full or closed.
func (r *LimitedRunner) Submit(task func()) error {
	var (
		canRun bool
		err    error
	)

	// The "add task" block.
	r.mu.Lock()
	if r.closed {
		// If closed.
		err = ErrClosed

	} else if r.c < r.maxConcurrency {
		// If not reach max concurrency, task can be run immediately.
		r.c += 1
		canRun = true

	} else if r.maxQueue < 0 || r.q.n < r.maxQueue {
		// If queue is unlimited or not full, enqueue the task.
		r.q.enqueue(task)

	} else {
		// Too busy now.
		err = ErrTooBusy

	}
	r.mu.Unlock()

	if !canRun {
		return err
	}

	go r.taskLoop(task)
	return nil
}

func (r *LimitedRunner) taskLoop(firstTask func()) {
	task := firstTask
	stop := false

	for !stop {
		// Run task.
		task()

		// The "done task" block.
		r.mu.Lock()
		if r.q.n > 0 {
			// If there are some queued tasks, dequeue one and run it directly.
			task = r.q.dequeue()

		} else {
			// Quit.
			r.c -= 1
			stop = true

			// Signal quit.
			r.cond.Signal()

		}
		r.mu.Unlock()
	}

}

// Close stops the runner and wait all ongoing and queued tasks to finish.
func (r *LimitedRunner) Close() {
	// The "close" block.
	r.mu.Lock()
	if !r.closed {
		// NOTE: At most one Close can run this branch of code.
		r.closed = true
		for r.c != 0 {
			r.cond.Wait()
		}
	}
	r.mu.Unlock()
}

func (q *taskQueue) enqueue(t func()) {
	node := &taskNode{
		task: t,
	}
	if q.n == 0 {
		q.head = node
		q.tail = node
		q.n = 1
	} else {
		q.tail.next = node
		q.tail = node
		q.n += 1
	}
}

func (q *taskQueue) dequeue() func() {
	node := q.head
	q.head = node.next
	q.n -= 1
	if q.n == 0 {
		q.tail = nil
	}

	node.next = nil
	return node.task
}
