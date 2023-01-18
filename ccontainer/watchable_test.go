package ccontainer

import (
	"context"
	"testing"
)

// TestWatchable tests the watchable ccontainer
func TestWatchable(t *testing.T) {
	ctx := context.Background()
	c := NewCContainer[*int](nil)

	errCh := make(chan error, 1)
	w := ToWatchable(c)
	_ = w.WaitValueEmpty(ctx, errCh) // should be instant

	var val = 5
	go c.SetValue(&val)
	gv, err := w.WaitValue(ctx, errCh)
	if err != nil {
		t.Fatal(err.Error())
	}
	if gv == nil || *gv != 5 {
		t.Fail()
	}
}
