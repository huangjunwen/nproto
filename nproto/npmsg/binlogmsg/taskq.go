package binlogmsg

import (
	"container/list"
	"errors"
	"sync"
)

var (
	errTaskQClosed = errors.New("taskQ is closed")
)

// taskQ is a simple multiple producer/SINGLE consumer task q.
type taskQ struct {
	ch chan struct{} // For notify only
	mu sync.Mutex
	ls *list.List
}

func newTaskQ() *taskQ {
	return &taskQ{
		ch: make(chan struct{}, 1),
		ls: list.New(),
	}
}

// non-blocking push item to the end of q
func (q *taskQ) Push(v interface{}) {
	if v == nil {
		panic(errors.New("Push nil"))
	}
	q.mu.Lock()
	q.ls.PushBack(v)
	q.mu.Unlock()
	select {
	case q.ch <- struct{}{}:
	default:
	}
}

// Pop wait and returns the first element of queue
func (q *taskQ) Pop() (v interface{}) {
POP:
	q.mu.Lock()
	elm := q.ls.Front()
	if elm != nil {
		v = q.ls.Remove(elm)
	}
	q.mu.Unlock()
	if v != nil {
		return
	}

	<-q.ch
	goto POP
}
