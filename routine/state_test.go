package routine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	protobuf_go_lite "github.com/aperturerobotics/protobuf-go-lite"
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

	k := NewStateRoutineContainerWithLogger[int](protobuf_go_lite.CompareComparable[int](), le)
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

func TestStateRoutineContainerRestartCompare(t *testing.T) {
	started := make(chan int, 2)
	stopped := make(chan int, 2)
	stateRoutine := func(ctx context.Context, state int) error {
		started <- state
		<-ctx.Done()
		stopped <- state
		return ctx.Err()
	}
	stateEqual := func(a, b int) bool { return a == b }
	routineEqual := func(a, b int) bool { return a/10 == b/10 }
	optionApplied := false
	container := NewStateRoutineContainerWithRestartCompare(
		stateEqual,
		routineEqual,
		newOption(func(*RoutineContainer) { optionApplied = true }),
	)
	if !optionApplied {
		t.Fatal("constructor option was not applied")
	}
	container.SetStateRoutine(stateRoutine)
	runCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	container.SetContext(runCtx, false)

	if _, changed, reset, running := container.SetState(11); !changed || reset || !running {
		t.Fatalf("initial state returned changed=%t reset=%t running=%t", changed, reset, running)
	}
	requireStateRoutineValue(t, started, 11, "initial routine start")

	noRestartBroadcast := stateRoutineBroadcast(container)
	waitReturn, changed, reset, running := container.SetState(12)
	if !changed || reset || !running || waitReturn != nil {
		t.Fatalf("equivalent routine state returned wait=%v changed=%t reset=%t running=%t", waitReturn != nil, changed, reset, running)
	}
	if got := container.GetState(); got != 12 {
		t.Fatalf("stored state = %d, want 12", got)
	}
	requireStateRoutineBroadcast(t, noRestartBroadcast, "equivalent routine state")

	unchangedBroadcast := stateRoutineBroadcast(container)
	waitReturn, changed, reset, running = container.SetState(12)
	if changed || reset || running || waitReturn != nil {
		t.Fatalf("unchanged state returned wait=%v changed=%t reset=%t running=%t", waitReturn != nil, changed, reset, running)
	}
	requireStateRoutineNoBroadcast(t, unchangedBroadcast, "unchanged state")

	restartBroadcast := stateRoutineBroadcast(container)
	waitReturn, changed, reset, running = container.SetState(21)
	if !changed || !reset || !running || waitReturn == nil {
		t.Fatalf("restart state returned wait=%v changed=%t reset=%t running=%t", waitReturn != nil, changed, reset, running)
	}
	requireStateRoutineBroadcast(t, restartBroadcast, "restart state")
	requireStateRoutineValue(t, stopped, 11, "restarted routine stop")
	requireStateRoutineValue(t, started, 21, "restarted routine start")

	cancel()
	requireStateRoutineValue(t, stopped, 21, "cleanup routine stop")
	container.ClearContext()
}

func TestStateRoutineContainerNilRestartComparePreservesRestart(t *testing.T) {
	started := make(chan int, 2)
	stopped := make(chan int, 2)
	container := NewStateRoutineContainerWithRestartCompare(
		func(a, b int) bool { return a == b },
		nil,
	)
	container.SetStateRoutine(func(ctx context.Context, state int) error {
		started <- state
		<-ctx.Done()
		stopped <- state
		return ctx.Err()
	})
	runCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	container.SetContext(runCtx, false)
	container.SetState(1)
	requireStateRoutineValue(t, started, 1, "initial routine start")

	waitReturn, changed, reset, running := container.SetState(2)
	if !changed || !reset || !running || waitReturn == nil {
		t.Fatalf("nil restart comparator returned wait=%v changed=%t reset=%t running=%t", waitReturn != nil, changed, reset, running)
	}
	requireStateRoutineValue(t, stopped, 1, "restarted routine stop")
	requireStateRoutineValue(t, started, 2, "restarted routine start")

	cancel()
	requireStateRoutineValue(t, stopped, 2, "cleanup routine stop")
	container.ClearContext()
}

func stateRoutineBroadcast[T comparable](container *StateRoutineContainer[T]) <-chan struct{} {
	var wait <-chan struct{}
	container.rc.bcast.HoldLock(func(_ func(), getWaitCh func() <-chan struct{}) {
		wait = getWaitCh()
	})
	return wait
}

func requireStateRoutineBroadcast(t *testing.T, wait <-chan struct{}, event string) {
	t.Helper()
	select {
	case <-wait:
	default:
		t.Fatalf("%s did not broadcast", event)
	}
}

func requireStateRoutineNoBroadcast(t *testing.T, wait <-chan struct{}, event string) {
	t.Helper()
	select {
	case <-wait:
		t.Fatalf("%s broadcast", event)
	default:
	}
}

func requireStateRoutineValue(t *testing.T, values <-chan int, want int, event string) {
	t.Helper()
	select {
	case got := <-values:
		if got != want {
			t.Fatalf("%s value = %d, want %d", event, got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not occur", event)
	}
}
