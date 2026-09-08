package refcount

import (
	"context"
	"testing"
	"time"
)

// TestInvalidateQueuedResolver preserves predecessor cleanup and resolves live
// references after invalidation cancels a resolver waiting for that cleanup.
func TestInvalidateQueuedResolver(t *testing.T) {
	// Hold the predecessor completion signal used to serialize resolvers.
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	predecessorDone := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-predecessorDone:
		default:
			close(predecessorDone)
		}
	})
	rc := NewRefCount[int](ctx, true, nil, nil, func(ctx context.Context, _ func()) (int, func(), error) {
		return 42, nil, ctx.Err()
	})
	t.Cleanup(rc.ClearContext)
	rc.mtx.Lock()
	rc.waitCh = predecessorDone
	rc.mtx.Unlock()

	// Queue a live reference and invalidate it before its predecessor finishes.
	result, ref := rc.AddRefPromise()
	t.Cleanup(ref.Release)
	rc.mtx.Lock()
	queuedDone := rc.waitCh
	rc.mtx.Unlock()
	if !rc.Invalidate() {
		t.Fatal("queued resolver was not invalidated")
	}

	// Cancellation cannot advertise completion ahead of predecessor cleanup.
	select {
	case <-queuedDone:
		t.Fatal("canceled resolver skipped unfinished predecessor cleanup")
	case <-time.After(20 * time.Millisecond):
	}
	close(predecessorDone)

	// Existing references receive the replacement resolver's result.
	value, err := result.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if value != 42 {
		t.Fatalf("resolved value = %d, want 42", value)
	}
}
