package csync

import (
	"context"
	"testing"
	"time"
)

// lockWaitContext reports when Lock has reached its context-aware wait.
type lockWaitContext struct {
	// Context supplies the actual cancellation signal.
	context.Context
	// waiting reports that Lock has captured the state-change watch.
	waiting chan struct{}
}

// Done observes entry to the real wait without altering cancellation behavior.
func (c lockWaitContext) Done() <-chan struct{} {
	// Report readiness without blocking repeated accesses to the context.
	select {
	case c.waiting <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

// TestRWMutexCanceledWriterWakesReaders retains an active reader throughout.
func TestRWMutexCanceledWriterWakesReaders(t *testing.T) {
	// Hold a reader so the writer queues and later readers must wait behind it.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var m RWMutex
	release, err := m.Lock(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	writerCtx, cancelWriter := context.WithCancel(ctx)
	defer cancelWriter()
	writerWaiting := make(chan struct{}, 1)
	writerDone := make(chan error, 1)
	go func() {
		unlock, err := m.Lock(lockWaitContext{writerCtx, writerWaiting}, true)
		if unlock != nil {
			unlock()
		}
		writerDone <- err
	}()
	<-writerWaiting

	// Observe the second reader's actual wait before canceling the writer.
	readerWaiting := make(chan struct{}, 16)
	readerDone := make(chan error, 16)
	for range 16 {
		go func() {
			unlock, err := m.Lock(lockWaitContext{ctx, readerWaiting}, false)
			if unlock != nil {
				unlock()
			}
			readerDone <- err
		}()
	}
	for range 16 {
		<-readerWaiting
	}
	cancelWriter()
	if err := <-writerDone; err != context.Canceled {
		t.Fatalf("writer cancellation: %v", err)
	}

	// A shared reader must proceed without waiting for the original reader to leave.
	for range 16 {
		if err := <-readerDone; err != nil {
			t.Fatalf("reader remained blocked after the writer left: %v", err)
		}
	}
}
