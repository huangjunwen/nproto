package task

import (
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
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	f := func() {
		wg.Done()
	}
	runner := NewLimitedTaskRunner(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := runner.Run(f)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Wait()
}
