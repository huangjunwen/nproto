package binlogmsg

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskQ(t *testing.T) {
	assert := assert.New(t)

	{
		q := newTaskQ()
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			v := q.Pop()
			assert.Equal(1, v)

			v = q.Pop()
			assert.Equal(2, v)
			wg.Done()
		}()

		time.Sleep(time.Second)
		q.Push(1)
		q.Push(2)
		wg.Wait()
	}

	{
		n := 1000
		q := newTaskQ()
		wg := &sync.WaitGroup{}
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				for j := 0; j < n; j++ {
					q.Push(1)
				}
				q.Push(0) // zero indicate end
				wg.Done()
			}()
		}

		zeros := 0
		ones := 0
		for {
			v := q.Pop().(int)
			if v == 0 {
				zeros++
			} else {
				ones++
			}
			if zeros >= n {
				break
			}
		}
		assert.Equal(n*n, ones)

	}

}
