package refcount

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aperturerobotics/util/ccontainer"
)

// TestRefCount tests the RefCount mechanism.
func TestRefCount(t *testing.T) {
	ctx := context.Background()
	target := ccontainer.NewCContainer[*string](nil)
	targetErr := ccontainer.NewCContainer[*error](nil)
	var valCalled, relCalled atomic.Bool
	rc := NewRefCount(nil, target, targetErr, func(ctx context.Context, released func()) (*string, func(), error) {
		val := "hello world"
		valCalled.Store(true)
		return &val, func() {
			relCalled.Store(true)
		}, nil
	})

	ref := rc.AddRef(nil)
	<-time.After(time.Millisecond * 50)
	if valCalled.Load() || relCalled.Load() {
		t.Fail()
	}

	rc.SetContext(ctx)
	<-time.After(time.Millisecond * 50)
	if !valCalled.Load() || relCalled.Load() {
		t.Fail()
	}

	var gotValue *string
	gotErr := rc.Access(ctx, func(val *string) error {
		gotValue = val
		return nil
	})

	waitVal, err := target.WaitValue(ctx, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	if waitVal != gotValue || gotErr != nil || relCalled.Load() {
		t.Fail()
	}
	ref.Release()

	if !relCalled.Load() {
		t.Fail()
	}
}

// TestRefCount_Released tests the RefCount released mechanism.
func TestRefCount_Released(t *testing.T) {
	ctx := context.Background()
	target := ccontainer.NewCContainer[*int](nil)
	targetErr := ccontainer.NewCContainer[*error](nil)
	var valCalled, relCalled atomic.Bool
	doCallRelease := make(chan struct{})
	ctr := 0
	rc := NewRefCount(nil, target, targetErr, func(ctx context.Context, released func()) (*int, func(), error) {
		valCalled.Store(true)
		ctr++
		val := ctr
		go func() {
			<-doCallRelease
			released()
		}()
		return &val, func() {
			relCalled.Store(true)
		}, nil
	})

	ref := rc.AddRef(nil)
	defer ref.Release()

	<-time.After(time.Millisecond * 50)
	if valCalled.Load() || relCalled.Load() {
		t.Fail()
	}

	rc.SetContext(ctx)
	<-time.After(time.Millisecond * 50)
	if !valCalled.Load() || relCalled.Load() {
		t.Fail()
	}

	var gotValue *int
	gotErr := rc.Access(ctx, func(val *int) error {
		gotValue = val
		return nil
	})
	if gotErr != nil {
		t.Fatal(gotErr.Error())
	}
	if *gotValue != ctr {
		t.Fatalf("expected value to be %v but had %v", ctr, *gotValue)
	}

	close(doCallRelease)
	<-time.After(time.Millisecond * 50)

	gotErr = rc.Access(ctx, func(val *int) error {
		gotValue = val
		return nil
	})
	if gotErr != nil {
		t.Fatal(gotErr.Error())
	}
	if ctr == 1 {
		t.Fail()
	}
	if *gotValue != ctr {
		t.Fatalf("expected value to be %v but had %v", ctr, *gotValue)
	}
}
