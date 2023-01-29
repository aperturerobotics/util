package routine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestRoutineContainer tests the routine container goroutine manager.
func TestRoutineContainer(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)
	vals := make(chan struct{})
	var exitWithErr atomic.Pointer[error]
	routineFn := func(ctx context.Context) error {
		if errPtr := exitWithErr.Load(); errPtr != nil {
			return *errPtr
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		case vals <- struct{}{}:
			return nil
		}
	}

	k := NewRoutineContainerWithLogger(le)
	if wasReset := k.SetRoutine(routineFn); wasReset {
		// expected !wasReset before context is set
		t.Fail()
	}

	// expect nothing to happen: context is unset.
	<-time.After(time.Millisecond * 10)
	select {
	case val := <-vals:
		t.Fatalf("unexpected value before set context: %s", val)
	default:
	}

	if !k.SetContext(ctx, true) {
		// expected to start with this call
		t.Fail()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 10)
	select {
	case <-vals:
	default:
		t.Fail()
	}

	// expect no extra value after
	<-time.After(time.Millisecond * 10)
	select {
	case <-vals:
		t.Fail()
	default:
	}

	// restart the routine
	if !k.RestartRoutine() {
		// expect it to be restarted
		t.Fail()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 10)
	select {
	case <-vals:
	default:
		t.Fail()
	}

	// unset context
	if !k.SetContext(nil, false) {
		// expect shutdown
		t.Fail()
	}

	// expect nothing happened (no difference)
	if k.SetContext(nil, false) {
		t.Fail()
	}

	// test wait exited
	var waitExitedReturned atomic.Pointer[error]
	startWaitExited := func() {
		go func() {
			err := k.WaitExited(ctx, nil)
			waitExitedReturned.Store(&err)
		}()
	}
	startWaitExited()

	<-time.After(time.Millisecond * 10)
	if waitExitedReturned.Load() != nil {
		t.Fail()
	}

	// set context
	if !k.SetContext(ctx, true) {
		t.Fail()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 10)
	if waitExitedReturned.Load() != nil {
		t.Fail()
	}
	select {
	case <-vals:
	default:
		t.Fail()
	}

	// set routine again
	if !k.SetRoutine(routineFn) {
		t.Fail()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 10)
	if waitExitedReturned.Load() != nil {
		t.Fail()
	}
	select {
	case <-vals:
	default:
		t.Fail()
	}

	// this time, tell the routine to fail
	expectedErr := errors.New("expected error for testing")
	exitWithErr.Store(&expectedErr)
	k.RestartRoutine()

	<-time.After(time.Millisecond * 10)
	errPtr := waitExitedReturned.Load()
	if errPtr == nil {
		t.Fail()
	} else if (*errPtr) != expectedErr {
		t.Fail()
	}

	exitWithErr.Store(nil)
	waitExitedReturned.Store(nil)
	startWaitExited()

	k.RestartRoutine()
	k.SetRoutine(nil)
	<-time.After(time.Millisecond * 10)
	errPtr = waitExitedReturned.Load()
	if errPtr == nil {
		t.Fail()
	} else if (*errPtr) != nil {
		t.Fail()
	}
}
