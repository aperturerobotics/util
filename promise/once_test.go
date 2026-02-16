package promise

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestOnce(t *testing.T) {
	t.Run("ResolveOnce", func(t *testing.T) {
		callCount := 0
		o := NewOnce(func(ctx context.Context) (int, error) {
			callCount++
			return 42, nil
		})

		ctx := context.Background()
		result, err := o.Resolve(ctx)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}

		// Call again, should return same result without calling the function
		result, err = o.Resolve(ctx)
		if err != nil {
			t.Fatalf("Unexpected error on second call: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42 on second call, got %d", result)
		}
		if callCount != 1 {
			t.Errorf("Expected callback to still be called once, got %d", callCount)
		}
	})

	t.Run("ResolveError", func(t *testing.T) {
		callCount := 0
		expectedError := errors.New("test error")
		o := NewOnce(func(ctx context.Context) (int, error) {
			callCount++
			return 0, expectedError
		})

		ctx := context.Background()
		_, err := o.Resolve(ctx)
		if err != expectedError {
			t.Fatalf("Expected error %v, got %v", expectedError, err)
		}
		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}

		// Call again, should retry
		_, err = o.Resolve(ctx)
		if err != expectedError {
			t.Fatalf("Expected error %v on second call, got %v", expectedError, err)
		}
		if callCount != 2 {
			t.Errorf("Expected callback to be called twice, got %d", callCount)
		}
	})

	t.Run("ResolveConcurrent", func(t *testing.T) {
		var mu sync.Mutex
		callCount := 0
		o := NewOnce(func(ctx context.Context) (int, error) {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			time.Sleep(10 * time.Millisecond) // Simulate some work
			return 42, nil
		})

		ctx := context.Background()
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				result, err := o.Resolve(ctx)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != 42 {
					t.Errorf("Expected 42, got %d", result)
				}
			})
		}
		wg.Wait()

		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}
	})

	t.Run("ResolveWithCanceledContext", func(t *testing.T) {
		o := NewOnce(func(ctx context.Context) (int, error) {
			return 42, nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		_, err := o.Resolve(ctx)
		if err != context.Canceled {
			t.Fatalf("Expected context.Canceled error, got %v", err)
		}
	})
}
