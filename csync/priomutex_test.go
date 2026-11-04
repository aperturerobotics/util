package csync

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"
)

// waitForPrioWaiters waits until n callers are queued on m.
func waitForPrioWaiters(m *PrioMutex, n int) {
	for {
		m.mtx.Lock()
		var queued int
		for _, class := range m.classes {
			queued += len(class.waiters)
		}
		m.mtx.Unlock()
		if queued == n {
			return
		}
		runtime.Gosched()
	}
}

// TestPrioMutex checks priority order, arrival order within a priority,
// Waiting, and that a canceled waiter leaves the queue.
func TestPrioMutex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Hold the lock so later callers queue.
	var m PrioMutex
	release, ok := m.TryLock()
	if !ok {
		t.Fatal("TryLock failed with mutex unlocked")
	}
	if _, ok := m.TryLock(); ok {
		t.Fatal("TryLock succeeded with mutex locked")
	}
	if m.Waiting(0) {
		t.Fatal("Waiting reported a waiter on an idle queue")
	}

	// Queue a canceled waiter at the highest priority, which must leave no
	// trace once its Lock returns.
	canceledCtx, cancelWaiter := context.WithCancel(ctx)
	canceled := make(chan error, 1)
	go func() {
		_, err := m.Lock(canceledCtx, 5)
		canceled <- err
	}()
	waitForPrioWaiters(&m, 1)
	if !m.Waiting(5) {
		t.Fatal("Waiting missed a queued waiter")
	}
	cancelWaiter()
	if err := <-canceled; err != context.Canceled {
		t.Fatalf("canceled Lock returned %v", err)
	}
	if m.Waiting(0) || len(m.classes) != 0 {
		t.Fatal("canceled waiter left its class queued")
	}

	// Queue low, low, high, mid: each records its grant and releases.
	var order []string
	var orderMtx sync.Mutex
	var wg sync.WaitGroup
	callers := []struct {
		name string
		prio int
	}{{"low1", 0}, {"low2", 0}, {"high", 2}, {"mid", 1}}
	for i, c := range callers {
		wg.Go(func() {
			rel, err := m.Lock(ctx, c.prio)
			if err != nil {
				t.Error(err)
				return
			}
			orderMtx.Lock()
			order = append(order, c.name)
			orderMtx.Unlock()
			rel()
		})
		waitForPrioWaiters(&m, i+1)
	}

	// Check Waiting reports the highest queued priority.
	for prio, want := range []bool{true, true, true, false} {
		if got := m.Waiting(prio); got != want {
			t.Fatalf("Waiting(%d) = %v, want %v", prio, got, want)
		}
	}

	// Release, twice to check a repeated release does nothing, and check the
	// grant order.
	release()
	release()
	wg.Wait()
	if want := []string{"high", "mid", "low1", "low2"}; !slices.Equal(order, want) {
		t.Fatalf("grant order %v, want %v", order, want)
	}
	if m.Waiting(0) || m.locked {
		t.Fatal("mutex not idle after every holder released")
	}
}

// TestPrioMutexCancelHandoff checks that a waiter canceled as the lock is
// handed to it passes the lock on instead of losing it.
func TestPrioMutexCancelHandoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for range 200 {
		// Queue a waiter that will be canceled, then one behind it.
		var m PrioMutex
		release, _ := m.TryLock()
		firstCtx, cancelFirst := context.WithCancel(ctx)
		first := make(chan func(), 1)
		go func() {
			rel, _ := m.Lock(firstCtx, 1)
			first <- rel
		}()
		waitForPrioWaiters(&m, 1)
		second := make(chan error, 1)
		go func() {
			rel, err := m.Lock(ctx, 0)
			if err == nil {
				rel()
			}
			second <- err
		}()
		waitForPrioWaiters(&m, 2)

		// End the first waiter's ctx, waking it on that case, and grant it the
		// lock before it leaves the queue, then release it if it won.
		cancelFirst()
		release()
		if rel := <-first; rel != nil {
			rel()
		}

		// The second waiter must still get the lock.
		if err := <-second; err != nil {
			t.Fatalf("second waiter: %v", err)
		}
		if m.locked {
			t.Fatal("handoff left the mutex locked")
		}
	}
}

// TestPrioMutexHammer checks mutual exclusion under contention across
// priorities.
func TestPrioMutexHammer(t *testing.T) {
	var m PrioMutex
	var holders int
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for range 500 {
				rel, err := m.Lock(context.Background(), g%3)
				if err != nil {
					t.Error(err)
					return
				}
				holders++
				if holders != 1 {
					t.Error("two holders at once")
				}
				holders--
				rel()
			}
		})
	}
	wg.Wait()
}
