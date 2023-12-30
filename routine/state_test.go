package routine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aperturerobotics/util/vtcompare"
	"github.com/sirupsen/logrus"
)

// TestStateRoutineContainer tests the routine container goroutine manager.
func TestStateRoutineContainer(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)
	vals := make(chan int)
	var exitWithErr atomic.Pointer[error]
	var waitReturn chan struct{}
	routineFn := func(ctx context.Context, st int) error {
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
		case vals <- st:
			return nil
		}
	}

	k := NewStateRoutineContainerWithLogger[int](vtcompare.CompareComparable[int](), le)
	if _, wasReset, running := k.SetStateRoutine(routineFn); wasReset || running {
		// expected !wasReset and !running before context is set
		t.FailNow()
	}

	// expect nothing to happen: context is unset.
	<-time.After(time.Millisecond * 50)
	select {
	case val := <-vals:
		t.Fatalf("unexpected value before set context: %v", val)
	default:
	}

	// expect nothing to happen: state is unset
	if k.SetContext(ctx, true) {
		t.FailNow()
	}

	// expect to start now
	if _, changed, reset, running := k.SetState(1); !changed || !running || reset {
		t.FailNow()
	}

	checkVal := func(expected int) {
		select {
		case nval := <-vals:
			if expected != 0 && nval != expected {
				t.Fatalf("expected value %v but got %v", nval, expected)
			}
		default:
			t.FailNow()
		}
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	checkVal(1)

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
	checkVal(1)

	// update state
	if _, changed, _, running := k.SetState(2); !changed || !running {
		t.FailNow()
	}

	// expect value to be pushed to vals
	<-time.After(time.Millisecond * 50)
	checkVal(2)

	// expect nothing happened (no difference)
	if _, changed, reset, running := k.SetState(2); changed || reset || running {
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

	<-time.After(time.Millisecond * 50)

	// test wait exited
	var waitExitedReturned atomic.Pointer[error]
	waitReturn = make(chan struct{})
	startWaitExited := func() {
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

	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() != nil {
		t.FailNow()
	}

	close(waitReturn)
	<-time.After(time.Millisecond * 50)
	checkVal(2)
	<-time.After(time.Millisecond * 50)
	if waitExitedReturned.Load() == nil {
		t.FailNow()
	}
}
