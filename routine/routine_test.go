package routine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aperturerobotics/util/backoff"
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
	var waitReturn chan struct{}
	routineFn := func(ctx context.Context) error {
		if errPtr := exitWithErr.Load(); errPtr != nil {
			return *errPtr
		}
		if waitReturn != nil {
			select {
			case <-ctx.Done():
				return context.Canceled
			case <-waitReturn:
			}
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		case vals <- struct{}{}:
			return nil
		}
	}

	k := NewRoutineContainerWithLogger(le)
	if _, wasReset := k.SetRoutine(routineFn); wasReset {
		// expected !wasReset before context is set
		t.FailNow()
	}

	// expect nothing to happen: context is unset.
	<-time.After(time.Millisecond * 50)
	select {
	case val := <-vals:
		t.Fatalf("unexpected value before set context: %s", val)
	default:
	}

	if !k.SetContext(ctx, true) {
		// expected to start with this call
		t.FailNow()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	select {
	case <-vals:
	default:
		t.FailNow()
	}

	// expect no extra value after
	<-time.After(time.Millisecond * 50)
	select {
	case <-vals:
		t.FailNow()
	default:
	}

	// restart the routine
	if !k.RestartRoutine() {
		// expect it to be restarted
		t.FailNow()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	select {
	case <-vals:
	default:
		t.FailNow()
	}

	// unset context
	if !k.SetContext(nil, false) {
		// expect shutdown
		t.FailNow()
	}

	// expect nothing happened (no difference)
	if k.SetContext(nil, false) {
		t.FailNow()
	}

	// test wait exited
	var waitExitedReturned atomic.Pointer[error]
	startWaitExited := func() {
		waitExitedReturned.Store(nil)
		go func() {
			err := k.WaitExited(ctx, false, nil)
			waitExitedReturned.Store(&err)
		}()
	}
	startWaitExited()

	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() != nil {
		t.FailNow()
	}

	// set context
	if !k.SetContext(ctx, true) {
		t.FailNow()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() != nil {
		t.FailNow()
	}
	select {
	case <-vals:
	default:
		t.FailNow()
	}
	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() == nil {
		t.FailNow()
	}

	// set routine again.
	// expect !wasReset since the routine already exited.
	waitReturn = make(chan struct{})
	if _, wasReset := k.SetRoutine(routineFn); wasReset {
		t.FailNow()
	}
	<-time.After(time.Millisecond * 50)

	// set routine again, expect reset since waitReturn was set (routine is running)
	startWaitExited()
	if _, wasReset := k.SetRoutine(routineFn); !wasReset {
		t.FailNow()
	}

	// expect value to be pushed to vals
	close(waitReturn)
	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() != nil {
		t.FailNow()
	}
	select {
	case <-vals:
	default:
		t.FailNow()
	}

	// this time, tell the routine to fail
	expectedErr := errors.New("expected error for testing")
	exitWithErr.Store(&expectedErr)
	startWaitExited()
	k.RestartRoutine()

	<-time.After(time.Millisecond * 50)
	errPtr := waitExitedReturned.Load()
	if errPtr == nil {
		t.FailNow()
	} else if (*errPtr) != expectedErr {
		t.FailNow()
	}
}

// TestRoutineContainer_WaitExited tests the routine container wait exited func.
func TestRoutineContainer_WaitExited(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	vals := make(chan struct{}, 1)
	routineFn := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return context.Canceled
		case vals <- struct{}{}:
			return nil
		}
	}

	k := NewRoutineContainerWithLogger(le)
	var waitExitedReturned atomic.Bool
	go func() {
		_ = k.WaitExited(ctx, false, nil)
		waitExitedReturned.Store(true)
	}()
	<-time.After(time.Millisecond * 500)
	if waitExitedReturned.Load() {
		t.FailNow()
	}
	if _, wasReset := k.SetRoutine(routineFn); wasReset {
		// expected !wasReset before context is set
		t.FailNow()
	}
	if !k.SetContext(ctx, true) {
		// expected to start with this call
		t.FailNow()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	select {
	case <-vals:
	default:
		t.FailNow()
	}
	if !waitExitedReturned.Load() {
		t.FailNow()
	}
}

// TestRoutineContainer_WithBackoff tests the routine container backoff.
func TestRoutineContainer_WithBackoff(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	vals := make(chan struct{}, 1)
	retErrs := 5
	routineFn := func(ctx context.Context) error {
		retErrs--
		if retErrs != 0 {
			return errors.New("returned error to test backoff")
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		case vals <- struct{}{}:
			return nil
		}
	}

	bo := (&backoff.Backoff{
		BackoffKind: backoff.BackoffKind_BackoffKind_EXPONENTIAL,
		Exponential: &backoff.Exponential{
			InitialInterval: 100,
			MaxInterval:     500,
		},
	}).Construct()
	k := NewRoutineContainer(WithBackoff(bo))
	if _, wasReset := k.SetRoutine(routineFn); wasReset {
		// expected !wasReset before context is set
		t.FailNow()
	}
	if !k.SetContext(ctx, true) {
		// expected to start with this call
		t.FailNow()
	}

	// expect backoffs to occur
	<-vals
}
