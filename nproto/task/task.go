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
	// Submit submits a task to run. It must return an error if the task can't be run.
	// The call must not block.
	Submit(task func()) error
}

type LimitedTaskRunner struct {
	maxConcurrency int // > 0
	maxQueue       int

	cond   *sync.Cond
	mu     sync.Mutex
	closed bool
	c      int       // current running go routines: c <= maxConcurrency
	q      taskQueue // current task queue: q.n <= maxQueue if maxQueue >= 0
}

type taskQueue struct {
	head *taskNode
	tail *taskNode
	n    int
}

type taskNode struct {
	t    func()
	next *taskNode
}

func NewLimitedTaskRunner(maxConcurrency, maxQueue int) *LimitedTaskRunner {
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	r := &LimitedTaskRunner{
		maxConcurrency: maxConcurrency,
		maxQueue:       maxQueue,
	}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func (r *LimitedTaskRunner) Submit(task func()) error {
	var (
		canRun bool
		err    error
	)

	r.mu.Lock()
	for {
		// Check closed.
		if r.closed {
			err = ErrClosed
			break
		}

		// If not reach max concurrency, t can be run immediately.
		if r.c < r.maxConcurrency {
			r.c += 1
			canRun = true
			break
		}

		// If queue is unlimited or not full, enqueue the task.
		if r.maxQueue < 0 || r.q.n < r.maxQueue {
			r.q.enqueue(task)
			break
		}

		// Too busy now.
		err = ErrTooBusy
		break
	}
	r.mu.Unlock()

	if !canRun {
		return err
	}

	go r.taskLoop(task)
	return nil
}

func (r *LimitedTaskRunner) taskLoop(firstTask func()) {
	task := firstTask
	stop := false

	for !stop {
		// Run task.
		task()

		// Check queue.
		r.mu.Lock()
		if r.q.n > 0 {
			// If there are some queued tasks, dequeue one and run it.
			task = r.q.dequeue()
		} else {
			// Quit.
			r.c -= 1
			stop = true
		}
		r.mu.Unlock()
	}

	// Wakes close.
	r.cond.Broadcast()

}

func (r *LimitedTaskRunner) Close() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		for r.c != 0 {
			r.cond.Wait()
		}
	}
	r.mu.Unlock()
}

func (q *taskQueue) enqueue(t func()) {
	node := &taskNode{
		t: t,
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
	return node.t
}
