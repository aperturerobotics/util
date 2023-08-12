package csync

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// adapted from src/sync/rwmutex_test.go in Go
func parallelReader(m *RWMutex, clocked, cunlock chan bool, cdone chan error) {
	rel, err := m.Lock(context.Background(), false)
	if err != nil {
		cdone <- err
		return
	}
	clocked <- true
	<-cunlock
	rel()
	rel() // call twice to test atomics (concurrency safety)
	cdone <- nil
}

func doTestParallelReaders(t *testing.T, numReaders, gomaxprocs int) {
	runtime.GOMAXPROCS(gomaxprocs)
	var m RWMutex
	clocked := make(chan bool)
	cunlock := make(chan bool)
	cdone := make(chan error)
	for i := 0; i < numReaders; i++ {
		go parallelReader(&m, clocked, cunlock, cdone)
	}
	// Wait for all parallel RLock()s to succeed.
	for i := 0; i < numReaders; i++ {
		<-clocked
	}
	for i := 0; i < numReaders; i++ {
		cunlock <- true
	}
	// Wait for the goroutines to finish.
	for i := 0; i < numReaders; i++ {
		if err := <-cdone; err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestParallelReaders(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	doTestParallelReaders(t, 1, 4)
	doTestParallelReaders(t, 3, 4)
	doTestParallelReaders(t, 4, 2)
}

func reader(rwm *RWMutex, num_iterations int, activity *int32, cdone chan error) {
	for i := 0; i < num_iterations; i++ {
		rel, err := rwm.Lock(context.Background(), false)
		if err != nil {
			cdone <- err
		}
		n := atomic.AddInt32(activity, 1)
		if n < 1 || n >= 10000 {
			rel()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		for i := 0; i < 100; i++ {
		}
		atomic.AddInt32(activity, -1)
		rel()
	}
	cdone <- nil
}

func writer(rwm *RWMutex, num_iterations int, activity *int32, cdone chan error) {
	for i := 0; i < num_iterations; i++ {
		rel, err := rwm.Lock(context.Background(), true)
		if err != nil {
			cdone <- err
		}
		n := atomic.AddInt32(activity, 10000)
		if n != 10000 {
			rel()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		for i := 0; i < 100; i++ {
		}
		atomic.AddInt32(activity, -10000)
		rel()
	}
	cdone <- nil
}

func HammerRWMutex(t *testing.T, gomaxprocs, numReaders, num_iterations int) {
	runtime.GOMAXPROCS(gomaxprocs)
	// Number of active readers + 10000 * number of active writers.
	var activity int32
	var rwm RWMutex
	cdone := make(chan error)
	go writer(&rwm, num_iterations, &activity, cdone)
	var i int
	for i = 0; i < numReaders/2; i++ {
		go reader(&rwm, num_iterations, &activity, cdone)
	}
	go writer(&rwm, num_iterations, &activity, cdone)
	for ; i < numReaders; i++ {
		go reader(&rwm, num_iterations, &activity, cdone)
	}
	// Wait for the 2 writers and all readers to finish.
	for i := 0; i < 2+numReaders; i++ {
		if err := <-cdone; err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestRWMutex(t *testing.T) {
	var m RWMutex

	rel, err := m.Lock(context.Background(), true)
	if err != nil {
		t.Fatal(err.Error())
	}
	if rel == nil {
		t.FailNow()
	}

	if _, ok := m.TryLock(true); ok {
		t.Fatalf("TryLock succeeded with mutex locked")
	}
	if _, ok := m.TryLock(false); ok {
		t.Fatalf("TryRLock succeeded with mutex locked")
	}
	rel()

	rel, ok := m.TryLock(true)
	if !ok {
		t.Fatalf("TryLock failed with mutex unlocked")
	}
	rel()

	rel, ok = m.TryLock(false)
	if !ok {
		t.Fatalf("TryRLock failed with mutex unlocked")
	}
	rel2, ok := m.TryLock(false)
	if !ok {
		t.Fatalf("TryRLock failed with mutex rlocked")
	}
	if _, ok := m.TryLock(true); ok {
		t.Fatalf("TryLock succeeded with mutex rlocked")
	}

	rel()
	rel2()

	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	n := 1000
	if testing.Short() {
		n = 5
	}
	HammerRWMutex(t, 1, 1, n)
	HammerRWMutex(t, 1, 3, n)
	HammerRWMutex(t, 1, 10, n)
	HammerRWMutex(t, 4, 1, n)
	HammerRWMutex(t, 4, 3, n)
	HammerRWMutex(t, 4, 10, n)
	HammerRWMutex(t, 10, 1, n)
	HammerRWMutex(t, 10, 3, n)
	HammerRWMutex(t, 10, 10, n)
	HammerRWMutex(t, 10, 5, n)
}

func TestRLocker(t *testing.T) {
	var wl RWMutex
	var rl sync.Locker
	wlocked := make(chan error, 1)
	rlocked := make(chan bool, 1)
	rl = wl.RLocker()
	n := 10
	var rel func()
	go func() {
		for i := 0; i < n; i++ {
			rl.Lock()
			rl.Lock()
			rlocked <- true
			var err error
			rel, err = wl.Lock(context.Background(), true)
			wlocked <- err
		}
	}()
	for i := 0; i < n; i++ {
		<-rlocked
		rl.Unlock()
		select {
		case <-wlocked:
			t.Fatal("RLocker() didn't read-lock it")
		default:
		}
		rl.Unlock()
		if err := <-wlocked; err != nil {
			t.Fatal(err.Error())
		}
		select {
		case <-rlocked:
			t.Fatal("RLocker() didn't respect the write lock")
		default:
		}
		rel()
	}
}

func benchmarkRWMutex(b *testing.B, localWork, writeRatio int) {
	var rwm RWMutex
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			foo++
			if foo%writeRatio == 0 {
				rel, err := rwm.Lock(ctx, true)
				if err != nil {
					b.Fatal(err.Error())
				}
				rel()
			} else {
				rel, err := rwm.Lock(ctx, false)
				if err != nil {
					b.Fatal(err.Error())
				}
				for i := 0; i != localWork; i += 1 {
					foo *= 2
					foo /= 2
				}
				rel()
			}
		}
		_ = foo
	})
}

func BenchmarkRWMutexWrite100(b *testing.B) {
	benchmarkRWMutex(b, 0, 100)
}

func BenchmarkRWMutexWrite10(b *testing.B) {
	benchmarkRWMutex(b, 0, 10)
}

func BenchmarkRWMutexWorkWrite100(b *testing.B) {
	benchmarkRWMutex(b, 100, 100)
}

func BenchmarkRWMutexWorkWrite10(b *testing.B) {
	benchmarkRWMutex(b, 100, 10)
}
