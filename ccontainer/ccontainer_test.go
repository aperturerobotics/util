package ccontainer

import (
	"context"
	"testing"
	"time"
)

// TestCContainer tests the concurrent container
func TestCContainer(t *testing.T) {
	ctx := context.Background()
	c := NewCContainer[*int](nil)

	errCh := make(chan error, 1)
	_ = c.WaitValueEmpty(ctx, errCh) // should be instant

	var val = 5
	go c.SetValue(&val)
	gv, err := c.WaitValue(ctx, errCh)
	if err != nil {
		t.Fatal(err.Error())
	}
	if gv == nil || *gv != 5 {
		t.Fail()
	}

	dl, dlCancel := context.WithDeadline(ctx, time.Now().Add(time.Millisecond*1))
	defer dlCancel()
	err = c.WaitValueEmpty(dl, errCh)
	if err != context.DeadlineExceeded {
		t.Fail()
	}

	c.SetValue(nil)
	_ = c.WaitValueEmpty(ctx, errCh) // should be instant

	swapPlusOne := func(val *int) *int {
		nv := 1
		if val != nil {
			nv = *val + 1
		}
		return &nv
	}

	for i := 1; i < 10; i++ {
		out := c.SwapValue(swapPlusOne)
		if out == nil || *out != i {
			t.Fail()
		}
	}
}

// TestCContainerWithEqual tests the concurrent container with an equal checker
func TestCContainerWithEqual(t *testing.T) {
	type data struct {
		value string
	}

	ctx := context.Background()
	c := NewCContainerWithEqual[*data](nil, func(a, b *data) bool {
		if (a == nil) != (b == nil) {
			return false
		}
		if b.value == "same" {
			return true
		}
		return a.value == b.value
	})

	mkInitial := func() *data {
		return &data{value: "hello"}
	}
	c.SetValue(mkInitial())

	var done chan struct{}
	start := func() {
		done = make(chan struct{})
		go func() {
			_, _ = c.WaitValueChange(ctx, mkInitial(), nil)
			close(done)
		}()
	}
	start()
	assertDone := func() {
		select {
		case <-done:
		case <-time.After(time.Millisecond * 100):
			t.Fatal("expected WaitValueChange to have returned")
		}
	}
	assertNotDone := func() {
		select {
		case <-done:
			t.Fatal("expected WaitValueChange to not return yet")
		case <-time.After(time.Millisecond * 50):
		}
	}
	assertNotDone()
	c.SetValue(mkInitial())
	assertNotDone()
	c.SetValue(&data{value: "same"})
	assertNotDone()
	c.SetValue(&data{value: "different"})
	assertDone()
	start()
	assertDone()
	c.SetValue(mkInitial())
	start()
	assertNotDone()
	c.SetValue(&data{value: "different"})
	assertDone()
}
