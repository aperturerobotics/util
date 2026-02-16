package csync

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
)

// adapted from src/sync/mutex_test.go in Go
// GOMAXPROCS=10 go test

func HammerMutex(m *Mutex, loops int, cdone chan bool) {
	for range loops {
		release, err := m.Lock(context.Background())
		if err != nil {
			panic(err)
		}
		release()
	}
	cdone <- true
}

func TestMutex(t *testing.T) {
	if n := runtime.SetMutexProfileFraction(1); n != 0 {
		t.Logf("got mutexrate %d expected 0", n)
	}
	defer runtime.SetMutexProfileFraction(0)

	m := new(Mutex)

	release, err := m.Lock(context.Background())
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	_, ok := m.TryLock()
	if ok {
		t.Fatalf("TryLock succeeded with mutex locked")
	}
	release()
	release2, ok := m.TryLock()
	if !ok {
		t.Fatalf("TryLock failed with mutex unlocked")
	}
	release2()

	c := make(chan bool)
	for range 10 {
		go HammerMutex(m, 1000, c)
	}
	for range 10 {
		<-c
	}
}

var misuseTests = []struct {
	name string
	f    func()
}{
	{
		"Mutex.Unlock",
		func() {
			var mu Mutex
			mu.Locker().Unlock()
		},
	},
	{
		"Mutex.Unlock2",
		func() {
			var mu Mutex
			release, _ := mu.Lock(context.Background())
			mu.Locker().Unlock()
			release()
		},
	},
}

func init() {
	if len(os.Args) == 3 && os.Args[1] == "TESTMISUSE" {
		for _, test := range misuseTests {
			if test.name == os.Args[2] {
				func() {
					defer func() { recover() }()
					test.f()
				}()
				fmt.Printf("test completed\n")
				os.Exit(0)
			}
		}
		fmt.Printf("unknown test\n")
		os.Exit(0)
	}
}

func BenchmarkMutexUncontended(b *testing.B) {
	type PaddedMutex struct {
		Mutex
		pad [128]uint8 //nolint:unused
	}
	b.RunParallel(func(pb *testing.PB) {
		var mu PaddedMutex
		for pb.Next() {
			release, err := mu.Lock(context.Background())
			if err != nil {
				b.Fatalf("Lock failed: %v", err)
			}
			release()
		}
	})
}

func benchmarkMutex(b *testing.B, slack, work bool) {
	var mu Mutex
	if slack {
		b.SetParallelism(10)
	}
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			release, err := mu.Lock(context.Background())
			if err != nil {
				b.Fatalf("Lock failed: %v", err)
			}
			release()
			if work {
				for range 100 {
					foo *= 2
					foo /= 2
				}
			}
		}
		_ = foo
	})
}

func BenchmarkMutex(b *testing.B) {
	benchmarkMutex(b, false, false)
}

func BenchmarkMutexSlack(b *testing.B) {
	benchmarkMutex(b, true, false)
}

func BenchmarkMutexWork(b *testing.B) {
	benchmarkMutex(b, false, true)
}

func BenchmarkMutexWorkSlack(b *testing.B) {
	benchmarkMutex(b, true, true)
}

func BenchmarkMutexNoSpin(b *testing.B) {
	// This benchmark models a situation where spinning in the mutex should be
	// non-profitable and allows to confirm that spinning does not do harm.
	// To achieve this we create excess of goroutines most of which do local work.
	// These goroutines yield during local work, so that switching from
	// a blocked goroutine to other goroutines is profitable.
	// As a matter of fact, this benchmark still triggers some spinning in the mutex.
	var m Mutex
	var acc0, acc1 uint64
	b.SetParallelism(4)
	b.RunParallel(func(pb *testing.PB) {
		c := make(chan bool)
		var data [4 << 10]uint64
		for i := 0; pb.Next(); i++ {
			if i%4 == 0 {
				release, err := m.Lock(context.Background())
				if err != nil {
					b.Fatalf("Lock failed: %v", err)
				}
				acc0 -= 100
				acc1 += 100
				release()
			} else {
				for i := 0; i < len(data); i += 4 {
					data[i]++
				}
				// Elaborate way to say runtime.Gosched
				// that does not put the goroutine onto global runq.
				go func() {
					c <- true
				}()
				<-c
			}
		}
	})
}

func BenchmarkMutexSpin(b *testing.B) {
	// This benchmark models a situation where spinning in the mutex should be
	// profitable. To achieve this we create a goroutine per-proc.
	// These goroutines access considerable amount of local data so that
	// unnecessary rescheduling is penalized by cache misses.
	var m Mutex
	var acc0, acc1 uint64
	b.RunParallel(func(pb *testing.PB) {
		var data [16 << 10]uint64
		for i := 0; pb.Next(); i++ {
			release, err := m.Lock(context.Background())
			if err != nil {
				b.Fatalf("Lock failed: %v", err)
			}
			acc0 -= 100
			acc1 += 100
			release()
			for i := 0; i < len(data); i += 4 {
				data[i]++
			}
		}
	})
}
