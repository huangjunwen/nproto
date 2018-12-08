package task

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func BenchmarkRawGoroutines(b *testing.B) {
	wg := &sync.WaitGroup{}
	f := func() {
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

func BenchmarkLimitedTaskRunner(b *testing.B) {
	bench := func(logMaxConcurrency uint) {
		b.Run(fmt.Sprintf("2**%d", logMaxConcurrency), func(b *testing.B) {
			maxConcurrency := 1 << logMaxConcurrency
			wg := &sync.WaitGroup{}
			wg.Add(b.N)
			f := func() {
				wg.Done()
			}
			runner := NewLimitedTaskRunner(int(maxConcurrency), -1)

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
	bench(4)
	bench(8)
	bench(10)
	bench(14)
}
