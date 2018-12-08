package task

import (
	"fmt"
	"log"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskQueue(t *testing.T) {
	assert := assert.New(t)

	q := taskQueue{}

	// NOTE: assert.Equal does not work on functions.
	// So here is some workaround.
	var f1, f2 func()
	var assertFn func(fn func(), expect int)
	{
		which := 0
		reset := func() { which = 0 }
		f1 = func() { which = 1 }
		f2 = func() { which = 2 }
		assertFn = func(fn func(), expect int) {
			reset()
			defer reset()
			fn()
			assert.Equal(expect, which)
		}
	}

	{
		q.enqueue(f1)
		assertFn(q.head.task, 1)
		assertFn(q.tail.task, 1)
		assert.Equal(1, q.n)
	}

	{
		q.enqueue(f2)
		assertFn(q.head.task, 1)
		assertFn(q.head.next.task, 2)
		assertFn(q.tail.task, 2)
		assert.Equal(2, q.n)
	}

	{
		f := q.dequeue()
		assertFn(f, 1)
		assertFn(q.head.task, 2)
		assertFn(q.tail.task, 2)
		assert.Equal(1, q.n)
	}

	{
		f := q.dequeue()
		assertFn(f, 2)
		assert.Nil(q.head)
		assert.Nil(q.tail)
		assert.Equal(0, q.n)
	}

}

func TestNoQueue(t *testing.T) {
	assert := assert.New(t)
	runner := NewDefaultTaskRunner(1, 0)

	assert.NoError(runner.Submit(func() {
		time.Sleep(100 * time.Microsecond)
	}))
	assert.Error(runner.Submit(func() {
		time.Sleep(101 * time.Microsecond)
	}))
}

func TestLimitQueue(t *testing.T) {
	assert := assert.New(t)
	runner := NewDefaultTaskRunner(1, 1)

	assert.NoError(runner.Submit(func() {
		time.Sleep(100 * time.Microsecond)
	}))
	assert.NoError(runner.Submit(func() {
		time.Sleep(101 * time.Microsecond)
	}))
	assert.Error(runner.Submit(func() {
		time.Sleep(102 * time.Microsecond)
	}))
}

func TestClose(t *testing.T) {
	assert := assert.New(t)
	runner := NewDefaultTaskRunner(2, -1)
	n := 10
	mu := &sync.Mutex{}
	remain := n
	f := func() {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		remain -= 1
		mu.Unlock()
	}

	for i := 0; i < n; i++ {
		assert.NoError(runner.Submit(f))
	}
	runner.Close()

	mu.Lock()
	assert.Equal(0, remain)
	mu.Unlock()

}

func BenchmarkRawGoroutines(b *testing.B) {
	wg := &sync.WaitGroup{}
	f := func() {
		time.Sleep(100 * time.Microsecond)
		wg.Done()
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go f()
	}
	wg.Wait()
}

func BenchmarkDefaultTaskRunner(b *testing.B) {
	bench := func(logMaxConcurrency int) {
		b.Run(fmt.Sprintf("10**%d", logMaxConcurrency), func(b *testing.B) {
			maxConcurrency := int(math.Pow10(logMaxConcurrency))
			wg := &sync.WaitGroup{}
			wg.Add(b.N)
			f := func() {
				time.Sleep(100 * time.Microsecond)
				wg.Done()
			}
			runner := NewDefaultTaskRunner(maxConcurrency, -1)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := runner.Submit(f)
				if err != nil {
					log.Fatal(err)
				}
			}
			wg.Wait()
		})
	}
	bench(0)
	bench(1)
	bench(2)
	bench(3)
	bench(4)
	bench(5)
}
