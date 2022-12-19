package promise

import (
	"context"
	"testing"
)

// TestPromiseContainer tests the PromiseContainer mechanics.
func TestPromiseContainer(t *testing.T) {
	ctx := context.Background()
	err := CheckPromiseLike(ctx, func() PromiseLike[int] {
		return NewPromise[int]()
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}
