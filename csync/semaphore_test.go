package csync

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// waitForSemaphoreWaiters waits until n callers are queued on s.
func waitForSemaphoreWaiters(s *Semaphore, n int) {
	for {
		s.mtx.Lock()
		queued := len(s.waiters)
		s.mtx.Unlock()
		if queued == n {
			return
		}
		runtime.Gosched()
	}
}

// acquireAsync starts Acquire of weight n and returns the channel its result
// arrives on.
func acquireAsync(ctx context.Context, s *Semaphore, n int64) chan error {
	done := make(chan error, 1)
	go func() { done <- s.Acquire(ctx, n) }()
	return done
}

// requireAdmitted fails unless done delivers a nil result.
func requireAdmitted(t *testing.T, name string, done chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s was not admitted", name)
	}
}

// requireWaiting fails if done has delivered a result.
func requireWaiting(t *testing.T, name string, done chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("%s returned early: %v", name, err)
	default:
	}
}

// TestSemaphore walks arrival order without barging, raising and lowering
// the limit, and a canceled head waiter.
func TestSemaphore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fill a limit of two.
	s := NewSemaphore(2)
	if err := s.Acquire(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if !s.TryAcquire(1) {
		t.Fatal("TryAcquire failed under the limit")
	}
	if s.TryAcquire(1) {
		t.Fatal("TryAcquire succeeded over the limit")
	}

	// Queue a large waiter, then a small one behind it.
	large := acquireAsync(ctx, s, 2)
	waitForSemaphoreWaiters(s, 1)
	small := acquireAsync(ctx, s, 1)
	waitForSemaphoreWaiters(s, 2)

	// A release leaves room for the small waiter only, which must not pass
	// the large one, nor may TryAcquire.
	s.Release(1)
	if s.TryAcquire(1) {
		t.Fatal("TryAcquire overtook a waiter")
	}
	requireWaiting(t, "small", small)
	requireWaiting(t, "large", large)

	// Raising the limit to three admits the large waiter alone.
	s.SetLimit(3)
	requireAdmitted(t, "large", large)
	requireWaiting(t, "small", small)

	// Lowering the limit to one holds the small waiter until both earlier
	// holders drain.
	s.SetLimit(1)
	s.Release(2)
	requireWaiting(t, "small", small)
	s.Release(1)
	requireAdmitted(t, "small", small)

	// A canceled head waiter leaves the queue and lets the next one in.
	s.SetLimit(2)
	headCtx, cancelHead := context.WithCancel(ctx)
	head := acquireAsync(headCtx, s, 2)
	waitForSemaphoreWaiters(s, 1)
	next := acquireAsync(ctx, s, 1)
	waitForSemaphoreWaiters(s, 2)
	cancelHead()
	if err := <-head; err != context.Canceled {
		t.Fatalf("canceled Acquire returned %v", err)
	}
	requireAdmitted(t, "next", next)
	s.Release(2)
	if s.held != 0 || len(s.waiters) != 0 {
		t.Fatalf("semaphore not idle: held %d, %d waiting", s.held, len(s.waiters))
	}
}

// TestSemaphoreCancelGrant checks that a waiter canceled as it is admitted
// returns its weight instead of leaking it.
func TestSemaphoreCancelGrant(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for range 200 {
		// Queue a waiter that will be canceled, then one behind it.
		s := NewSemaphore(1)
		s.TryAcquire(1)
		firstCtx, cancelFirst := context.WithCancel(ctx)
		first := acquireAsync(firstCtx, s, 1)
		waitForSemaphoreWaiters(s, 1)
		second := acquireAsync(ctx, s, 1)
		waitForSemaphoreWaiters(s, 2)

		// End the first waiter's ctx, waking it on that case, and admit it
		// before it leaves the queue, then release it if it won.
		cancelFirst()
		s.Release(1)
		if err := <-first; err == nil {
			s.Release(1)
		}

		// The second waiter must still be admitted.
		requireAdmitted(t, "second", second)
		s.Release(1)
		if s.held != 0 {
			t.Fatalf("weight leaked: held %d", s.held)
		}
	}
}
