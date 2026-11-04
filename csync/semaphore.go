package csync

import (
	"context"
	"slices"
	"sync"
)

// Semaphore implements a weighted semaphore that accepts a Context and whose
// limit can change while it is in use.
//
// Waiters are admitted in arrival order: a caller that cannot be admitted at
// once queues behind every earlier waiter, even when its own weight would
// fit, so a large request is not starved by smaller ones that arrive later.
// A release or a raised limit wakes only the waiters it admits. A lowered
// limit admits nobody until the callers already holding weight release
// enough of it to fit. A waiter whose weight exceeds the limit blocks the
// waiters behind it until the limit is raised or its ctx ends.
//
// An empty value Semaphore{} has a limit of zero and admits nobody until
// SetLimit raises it.
type Semaphore struct {
	// mtx guards the fields below and is never held across a wait.
	mtx sync.Mutex
	// limit is the most weight admitted callers may hold together.
	limit int64
	// held is the weight admitted callers hold. It exceeds limit after the
	// limit is lowered, until they release enough of it.
	held int64
	// waiters is the queue of callers waiting for weight, oldest first.
	waiters []*semaphoreWaiter
}

// semaphoreWaiter is one caller waiting in Acquire.
type semaphoreWaiter struct {
	// n is the weight the caller waits for.
	n int64
	// ready closes when the semaphore admits the caller, with n already
	// counted in held.
	ready chan struct{}
}

// NewSemaphore constructs a Semaphore admitting up to limit weight at once.
func NewSemaphore(limit int64) *Semaphore {
	return &Semaphore{limit: limit}
}

// Acquire waits until the Semaphore admits weight n. Returns ctx's error if
// ctx ends before then, leaving the Semaphore unchanged. A caller admitted
// with Acquire or TryAcquire returns the weight with Release.
func (s *Semaphore) Acquire(ctx context.Context, n int64) error {
	// Take the weight when no caller waits ahead and it fits, otherwise
	// queue for it.
	s.mtx.Lock()
	if len(s.waiters) == 0 && s.held+n <= s.limit {
		s.held += n
		s.mtx.Unlock()
		return nil
	}
	w := &semaphoreWaiter{n: n, ready: make(chan struct{})}
	s.waiters = append(s.waiters, w)
	s.mtx.Unlock()

	// Wait for a release or a raised limit to admit the caller.
	select {
	case <-w.ready:
		return nil
	case <-ctx.Done():
	}

	// Leave the queue, returning weight granted during cancellation. Either
	// change can admit the waiters behind this one.
	s.mtx.Lock()
	if i := slices.Index(s.waiters, w); i >= 0 {
		s.waiters = slices.Delete(s.waiters, i, i+1)
	} else {
		s.held -= n
	}
	s.grantLocked()
	s.mtx.Unlock()
	return ctx.Err()
}

// TryAcquire takes weight n when no caller waits and it fits under the
// limit. Returns false, leaving the Semaphore unchanged, otherwise.
func (s *Semaphore) TryAcquire(n int64) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if len(s.waiters) != 0 || s.held+n > s.limit {
		return false
	}
	s.held += n
	return true
}

// Release returns weight n taken with Acquire or TryAcquire and admits the
// waiters that now fit. Panics if more weight is released than is held.
func (s *Semaphore) Release(n int64) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.held -= n
	if s.held < 0 {
		panic("csync: Semaphore released more than held")
	}
	s.grantLocked()
}

// SetLimit changes the most weight admitted callers may hold together. A
// raised limit admits the waiters that now fit. A lowered limit takes effect
// as admitted callers release; no caller loses weight it holds.
func (s *Semaphore) SetLimit(limit int64) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.limit = limit
	s.grantLocked()
}

// grantLocked admits waiters in arrival order while the oldest fits under
// the limit. The caller holds mtx.
func (s *Semaphore) grantLocked() {
	// Admit the oldest waiters that fit.
	var i int
	for i < len(s.waiters) && s.held+s.waiters[i].n <= s.limit {
		s.held += s.waiters[i].n
		close(s.waiters[i].ready)
		s.waiters[i] = nil
		i++
	}

	// Drop the admitted waiters from the queue.
	s.waiters = s.waiters[i:]
	if len(s.waiters) == 0 {
		s.waiters = nil
	}
}
