package promise

import (
	"context"
	"testing"
)

// TestPromise tests the Promise mechanics.
func TestPromise(t *testing.T) {
	ctx := context.Background()
	err := CheckPromiseLike(ctx, func() PromiseLike[int] {
		return NewPromise[int]()
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}
