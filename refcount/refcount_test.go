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

	firstRef := ref
	prom, ref := rc.AddRefPromise()
	// release the first ref after adding the second
	firstRef.Release()
	val, err := prom.Await(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	if (*val) != "hello world" {
		t.Fail()
	}

	waitVal, err := target.WaitValue(ctx, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	if waitVal != val || relCalled.Load() {
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
	ctr := 0
	var relFunc func()
	rc := NewRefCount(nil, target, targetErr, func(ctx context.Context, released func()) (*int, func(), error) {
		valCalled.Store(true)
		ctr++
		val := ctr
		relFunc = released
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

	var v1 *int
	gotErr := rc.Access(ctx, func(ctx context.Context, val *int) error {
		v1 = val
		return nil
	})
	if gotErr != nil {
		t.Fatal(gotErr.Error())
	}
	if *v1 != ctr {
		t.Fatalf("expected value to be %v but had %v", ctr, *v1)
	}

	relFunc()
	<-time.After(time.Millisecond * 50)

	var v2 *int
	gotErr = rc.Access(ctx, func(ctx context.Context, val *int) error {
		v2 = val
		return nil
	})
	if gotErr != nil {
		t.Fatal(gotErr.Error())
	}
	if ctr == 1 {
		t.Fail()
	}
	if v2 == nil {
		t.Fatalf("expected value to be %v but got nil", ctr)
	}
	if *v2 != ctr {
		t.Fatalf("expected value to be %v but had %v", ctr, *v2)
	}
}

// TestRefCount_WaitWithReleased tests the RefCount wait with released mechanism.
func TestRefCount_WaitWithReleased(t *testing.T) {
	ctx := context.Background()
	doCallReleased := make(chan struct{})
	rc := NewRefCount(nil, nil, nil, func(ctx context.Context, released func()) (*bool, func(), error) {
		go func() {
			<-doCallReleased
			released()
		}()
		ret := true
		return &ret, func() {}, nil
	})

	var releasedCalled atomic.Bool
	valProm, ref, err := rc.WaitWithReleased(ctx, func() {
		if releasedCalled.Swap(true) {
			t.Fatal("released was called multiple times")
		}
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer ref.Release()

	rc.SetContext(ctx)
	val, err := valProm.Await(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	if val == nil || *val != true {
		t.Fail()
	}
	if releasedCalled.Load() {
		t.Fail()
	}
	close(doCallReleased)
	<-time.After(time.Millisecond * 50)
	if !releasedCalled.Load() {
		t.Fail()
	}
	ref.Release()
}
