package task

import (
	"context"
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

	subBenchmark := func(maxConcurrency int) {
		b.Run(fmt.Sprintf("%d", maxConcurrency), func(b *testing.B) {
			wg := &sync.WaitGroup{}
			f := func() {
				wg.Done()
			}
			runner := NewLimitedTaskRunner(maxConcurrency)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				wg.Add(1)
				err := runner.Run(context.Background(), f)
				if err != nil {
					log.Fatal(err)
				}
			}
			wg.Wait()
		})
	}

	subBenchmark(1)
	subBenchmark(10)
	subBenchmark(100)
	subBenchmark(1000)
	subBenchmark(10000)
	subBenchmark(100000)
	subBenchmark(1000000)
}
