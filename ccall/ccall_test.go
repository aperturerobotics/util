package ccall

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// TestCallConcurrently_Success tests calling multiple functions concurrently successfully.
func TestCallConcurrently_Success(t *testing.T) {
	var accum atomic.Int32

	var fns []CallConcurrentlyFunc
	for i := range int32(10) {
		x := i // copy value
		fns = append(fns, func(ctx context.Context) error {
			accum.Add(x)
			return nil
		})
	}

	if err := CallConcurrently(context.Background(), fns...); err != nil {
		t.Fatal(err.Error())
	}

	if val := accum.Load(); val != 45 {
		t.Fatalf("expected 45 but got %d", val)
	}
}

// TestCallConcurrently_Err tests calling multiple functions with an error.
func TestCallConcurrently_Err(t *testing.T) {
	errRet := errors.New("test error")

	var fns []CallConcurrentlyFunc
	for i := range 10 {
		i := i
		fns = append(fns, func(ctx context.Context) error {
			if i == 5 || i == 8 {
				return errRet
			}
			return nil
		})
	}

	if err := CallConcurrently(context.Background(), fns...); err != errRet {
		t.Fatalf("expected error but got %v", err)
	}
}
