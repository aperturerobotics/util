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
	rc := NewRefCount(nil, false, target, targetErr, func(ctx context.Context, released func()) (*string, func(), error) {
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
	rc := NewRefCount(nil, false, target, targetErr, func(ctx context.Context, released func()) (*int, func(), error) {
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
	rc := NewRefCount(nil, false, nil, nil, func(ctx context.Context, released func()) (*bool, func(), error) {
		go func() {
			<-doCallReleased
			released()
		}()
		ret := true
		return &ret, func() {}, nil
	})

	var releasedCalled atomic.Bool
	valProm, ref := rc.WaitWithReleased(ctx, func() {
		if releasedCalled.Swap(true) {
			t.Fatal("released was called multiple times")
		}
	})
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

// TestRefCount_KeepUnref tests the RefCount keep unreferenced flag.
func TestRefCount_KeepUnref(t *testing.T) {
	ctx := context.Background()
	target := ccontainer.NewCContainer[*int](nil)
	targetErr := ccontainer.NewCContainer[*error](nil)
	var valCalled, relCalled atomic.Bool
	ctr := 0
	var relFunc func()
	rc := NewRefCount(nil, true, target, targetErr, func(ctx context.Context, released func()) (*int, func(), error) {
		valCalled.Store(true)
		ctr++
		val := ctr
		relFunc = released
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

	ref.Release()
	<-time.After(time.Millisecond * 50)
	if relCalled.Load() {
		t.Fail()
	}

	valCalled.Store(false)
	prom, ref := rc.AddRefPromise()
	<-time.After(time.Millisecond * 50)
	if valCalled.Load() || relCalled.Load() {
		t.Fail()
	}
	val, err := prom.Await(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	if *val != 1 {
		t.Fail()
	}

	relFunc()
	<-time.After(time.Millisecond * 50)
	if !relCalled.Load() {
		t.Fail()
	}
	ref.Release()
	<-time.After(time.Millisecond * 50)
	prom, ref = rc.AddRefPromise()
	val, err = prom.Await(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
	if *val != 2 {
		t.Fail()
	}
	ref.Release()
}

// TestRefCount_Invalidate tests explicit invalidation and re-resolution.
func TestRefCount_Invalidate(t *testing.T) {
	ctx := context.Background()
	nextVal := 1
	rc := NewRefCount(nil, true, nil, nil, func(ctx context.Context, released func()) (*int, func(), error) {
		val := nextVal
		nextVal++
		return &val, nil, nil
	})
	rc.SetContext(ctx)

	first, firstRel, err := rc.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || *first != 1 {
		t.Fatalf("expected initial value 1, got %v", first)
	}

	if !rc.Invalidate() {
		t.Fatal("expected invalidate to report a change")
	}
	second, secondRel, err := rc.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second == nil || *second != 2 {
		t.Fatalf("expected re-resolved value 2, got %v", second)
	}
	firstRel()
	secondRel()
}

// TestRefCount_InvalidateCoalescesInflight tests repeated invalidates collapse
// into a single retry while a resolve is in flight.
func TestRefCount_InvalidateCoalescesInflight(t *testing.T) {
	ctx := context.Background()
	startedCh := make(chan int, 4)
	var calls atomic.Int32
	rc := NewRefCount(nil, true, nil, nil, func(ctx context.Context, released func()) (*int, func(), error) {
		n := int(calls.Add(1))
		startedCh <- n
		if n == 1 {
			<-ctx.Done()
			return nil, nil, ctx.Err()
		}
		val := n
		return &val, nil, nil
	})
	rc.SetContext(ctx)

	ref := rc.AddRef(nil)
	defer ref.Release()

	select {
	case n := <-startedCh:
		if n != 1 {
			t.Fatalf("expected first resolve to start first, got %d", n)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first resolve")
	}

	if !rc.Invalidate() {
		t.Fatal("expected first invalidate to queue a retry")
	}
	if rc.Invalidate() {
		t.Fatal("expected second invalidate to coalesce with the pending retry")
	}

	select {
	case n := <-startedCh:
		if n != 2 {
			t.Fatalf("expected second resolve after coalesced invalidation, got %d", n)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for retried resolve")
	}

	select {
	case n := <-startedCh:
		t.Fatalf("unexpected extra resolve started: %d", n)
	case <-time.After(100 * time.Millisecond):
	}

	val, rel, err := rc.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val == nil || *val != 2 {
		t.Fatalf("expected resolved value 2 after coalesced retry, got %v", val)
	}
	rel()
}

// TestRefCount_ResolveAsRefCount tests using a RefCount.Resolve as the resolver for another RefCount.
func TestRefCount_ResolveAsRefCount(t *testing.T) {
	ctx := context.Background()
	doCallReleased := make(chan struct{})
	rc := NewRefCount(nil, false, nil, nil, func(ctx context.Context, released func()) (*bool, func(), error) {
		go func() {
			<-doCallReleased
			released()
		}()
		ret := true
		return &ret, func() {}, nil
	})

	rc2 := NewRefCount(nil, false, nil, nil, rc.ResolveWithReleased)

	var releasedCalled atomic.Bool
	valProm, ref := rc2.WaitWithReleased(ctx, func() {
		if releasedCalled.Swap(true) {
			t.Fatal("released was called multiple times")
		}
	})
	defer ref.Release()

	rc.SetContext(ctx)
	rc2.SetContext(ctx)
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
