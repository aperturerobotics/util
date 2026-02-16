package keyed

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aperturerobotics/util/backoff"
	"github.com/sirupsen/logrus"
)

// testData contains some test metadata.
type testData struct {
	value string
}

// TestKeyed tests the keyed goroutine manager.
func TestKeyed(t *testing.T) {
	ctx := context.Background()
	vals := make(chan string, 10)
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)
	k := NewKeyed(func(key string) (Routine, *testData) {
		return func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return context.Canceled
			case vals <- key:
				return nil
			}
		}, &testData{}
	}, WithExitLogger[string, *testData](le))

	nsend := 101
	keys := make([]string, nsend)
	for i := range nsend {
		key := "routine-" + strconv.Itoa(i)
		keys[i] = key
	}

	added, removed := k.SyncKeys(keys, false)
	if len(removed) != 0 || !slices.Equal(added, keys) {
		t.FailNow()
	}

	nsend--
	keys = keys[:nsend]
	added, removed = k.SyncKeys(keys, false)
	if len(removed) != 1 || len(added) != 0 {
		t.FailNow()
	}

	// expect nothing to have been pushed to vals yet
	<-time.After(time.Millisecond * 10)
	select {
	case val := <-vals:
		t.Fatalf("unexpected value before set context: %s", val)
	default:
	}

	// start execution
	k.SetContext(ctx, false)

	seen := make(map[string]struct{})
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err().Error())
		case val := <-vals:
			if _, ok := seen[val]; ok {
				t.Fatalf("duplicate value: %s", val)
			}
			seen[val] = struct{}{}
			if len(seen) == nsend {
				// success
				return
			}
		}
	}
}

// TestKeyed_WithDelay tests the delay removing unreferenced keys.
func TestKeyed_WithDelay(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	var called, canceled atomic.Bool
	calledCh := make(chan struct{})
	canceledCh := make(chan struct{})

	k := NewKeyed(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				called.Store(true)
				close(calledCh)
				<-ctx.Done()
				canceled.Store(true)
				close(canceledCh)
				return nil
			}, &testData{}
		},
		WithExitLogger[string, *testData](le),
		WithReleaseDelay[string, *testData](time.Millisecond*180),
	)

	// start execution
	k.SetContext(ctx, false)

	k.SetKey("test", true)
	<-calledCh
	if !called.Load() || canceled.Load() {
		t.Fail()
	}

	// Remove the key, but it should still be running due to delay
	_ = k.RemoveKey("test")

	// Create a timer to check if the routine is still running after some time
	// This is one case where we need a timer since we're testing time-based behavior
	timer := time.NewTimer(time.Millisecond * 100)
	select {
	case <-canceledCh:
		t.Fatal("routine should not have been canceled yet")
	case <-timer.C:
		// Expected - routine should still be running
	}

	// Now wait for cancellation to happen after the delay
	<-canceledCh
	if !called.Load() || !canceled.Load() {
		t.Fail()
	}

	// Reset for second test
	canceled.Store(false)
	called.Store(false)
	calledCh = make(chan struct{})
	canceledCh = make(chan struct{})

	k.SetKey("test", false)
	<-calledCh
	if !called.Load() || canceled.Load() {
		t.Fail()
	}

	// Remove the key, but it should still be running due to delay
	_ = k.RemoveKey("test")

	// Set the key again before the delay expires
	k.SetKey("test", false)

	// Verify the routine is still running and wasn't canceled
	timer.Reset(time.Millisecond * 200)
	select {
	case <-canceledCh:
		t.Fatal("routine should not have been canceled")
	case <-timer.C:
		// Expected - routine should still be running
	}

	if !called.Load() || canceled.Load() {
		t.Fail()
	}
}

// TestKeyedWithRetry tests the keyed goroutine manager.
func TestKeyedWithRetry(t *testing.T) {
	ctx := context.Background()
	vals := make(chan string, 10)
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)
	i := 5
	k := NewKeyed(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				if i == 0 {
					select {
					case <-ctx.Done():
						return context.Canceled
					case vals <- key:
						return nil
					}
				}
				i--
				return errors.New("returning error to test retry")
			}, &testData{}
		},
		WithExitLogger[string, *testData](le),
		WithRetry[string, *testData](&backoff.Backoff{
			BackoffKind: backoff.BackoffKind_BackoffKind_EXPONENTIAL,
			Exponential: &backoff.Exponential{
				InitialInterval:     200,
				MaxInterval:         1000,
				RandomizationFactor: 0,
			},
		}),
	)

	k.SetContext(ctx, true)
	_, existed := k.SetKey("test-key", true)
	if existed {
		t.FailNow()
	}

	val := <-vals
	if val != "test-key" {
		t.FailNow()
	}
	if i != 0 {
		t.FailNow()
	}
}

// TestKeyedRefCount tests the reference counting functionality.
func TestKeyedRefCount(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	startCount := atomic.Int32{}
	stopCount := atomic.Int32{}

	k := NewKeyedRefCountWithLogger(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				startCount.Add(1)
				<-ctx.Done()
				stopCount.Add(1)
				return context.Canceled
			}, &testData{value: key}
		},
		le,
	)

	k.SetContext(ctx, false)

	// Add multiple references to the same key
	ref1, data1, existed1 := k.AddKeyRef("test-key")
	if existed1 {
		t.Fatal("key should not exist yet")
	}
	if data1.value != "test-key" {
		t.Fatal("unexpected data value")
	}

	ref2, data2, existed2 := k.AddKeyRef("test-key")
	if !existed2 {
		t.Fatal("key should exist now")
	}
	if data2.value != "test-key" {
		t.Fatal("unexpected data value")
	}

	// Create a channel to wait for the routine to start
	startCh := make(chan struct{})

	// Wait for the routine to start
	for range 100 {
		if startCount.Load() == 1 {
			close(startCh)
			break
		}
		// Small yield to allow other goroutines to run
		runtime.Gosched()
	}

	<-startCh
	if startCount.Load() != 1 {
		t.Fatal("routine should have started once")
	}
	if stopCount.Load() != 0 {
		t.Fatal("routine should not have stopped")
	}

	// Release one reference, routine should still be running
	ref1.Release()

	// Verify state hasn't changed
	if startCount.Load() != 1 {
		t.Fatal("routine should have started once")
	}
	if stopCount.Load() != 0 {
		t.Fatal("routine should not have stopped")
	}

	// Release the second reference, routine should stop
	ref2.Release()

	// Wait for the routine to stop
	stopCh := make(chan struct{})
	for range 100 {
		if stopCount.Load() == 1 {
			close(stopCh)
			break
		}
		runtime.Gosched()
	}

	<-stopCh
	if startCount.Load() != 1 {
		t.Fatal("routine should have started once")
	}
	if stopCount.Load() != 1 {
		t.Fatal("routine should have stopped")
	}

	// Add a reference again, routine should restart
	ref3, _, _ := k.AddKeyRef("test-key")

	// Wait for the routine to start again
	startCh2 := make(chan struct{})
	for range 100 {
		if startCount.Load() == 2 {
			close(startCh2)
			break
		}
		runtime.Gosched()
	}

	<-startCh2
	if startCount.Load() != 2 {
		t.Fatal("routine should have started twice")
	}
	if stopCount.Load() != 1 {
		t.Fatal("routine should have stopped once")
	}

	// Remove the key directly, should stop the routine
	k.RemoveKey("test-key")

	// Wait for the routine to stop again
	stopCh2 := make(chan struct{})
	for range 100 {
		if stopCount.Load() == 2 {
			close(stopCh2)
			break
		}
		runtime.Gosched()
	}

	<-stopCh2
	if startCount.Load() != 2 {
		t.Fatal("routine should have started twice")
	}
	if stopCount.Load() != 2 {
		t.Fatal("routine should have stopped twice")
	}

	// Releasing the reference after removal should be a no-op
	ref3.Release()
}

// TestExitCallbacks tests the exit callback functionality.
func TestExitCallbacks(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	var exitKey string
	var exitErr error
	var callbackCalled atomic.Bool
	callbackCh := make(chan struct{})

	exitCb := func(key string, routine Routine, data *testData, err error) {
		exitKey = key
		exitErr = err
		callbackCalled.Store(true)
		close(callbackCh)
	}

	k := NewKeyed(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				return errors.New("test error")
			}, &testData{}
		},
		WithExitLogger[string, *testData](le),
		WithExitCb(exitCb),
	)

	k.SetContext(ctx, true)
	_, existed := k.SetKey("test-key", true)
	if existed {
		t.Fatal("key should not exist yet")
	}

	// Wait for callback to be called
	<-callbackCh
	if !callbackCalled.Load() {
		t.Fatal("exit callback should have been called")
	}
	if exitKey != "test-key" {
		t.Fatal("wrong exit key")
	}
	if exitErr == nil || exitErr.Error() != "test error" {
		t.Fatal("wrong exit error")
	}
}

// TestRestartReset tests the restart and reset functionality.
func TestRestartReset(t *testing.T) {
	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	startCount := atomic.Int32{}
	resetCount := atomic.Int32{}
	startCh := make(chan struct{})

	k := NewKeyed(
		func(key string) (Routine, *testData) {
			resetCount.Add(1)
			return func(ctx context.Context) error {
				count := startCount.Add(1)
				if count == 1 {
					close(startCh)
				}
				<-ctx.Done()
				return context.Canceled
			}, &testData{value: key + "-" + strconv.Itoa(int(resetCount.Load()))}
		},
		WithExitLogger[string, *testData](le),
	)

	k.SetContext(ctx, false)
	_, existed := k.SetKey("test-key", true)
	if existed {
		t.Fatal("key should not exist yet")
	}

	<-startCh
	if startCount.Load() != 1 {
		t.Fatal("routine should have started once")
	}
	if resetCount.Load() != 1 {
		t.Fatal("constructor should have been called once")
	}

	// Test restart
	startCh2 := make(chan struct{})
	var startWg sync.WaitGroup
	startWg.Go(func() {
		for {
			if startCount.Load() == 2 {
				close(startCh2)
				return
			}
			runtime.Gosched()
		}
	})

	existed, restarted := k.RestartRoutine("test-key")
	if !existed || !restarted {
		t.Fatal("restart should have succeeded")
	}

	<-startCh2
	startWg.Wait()
	if startCount.Load() != 2 {
		t.Fatal("routine should have started twice")
	}
	if resetCount.Load() != 1 {
		t.Fatal("constructor should still have been called once")
	}

	// Test reset
	startCh3 := make(chan struct{})
	var startWg2 sync.WaitGroup
	startWg2.Go(func() {
		for {
			if startCount.Load() == 3 {
				close(startCh3)
				return
			}
			runtime.Gosched()
		}
	})

	existed, reset := k.ResetRoutine("test-key")
	if !existed || !reset {
		t.Fatal("reset should have succeeded")
	}

	<-startCh3
	startWg2.Wait()
	if startCount.Load() != 3 {
		t.Fatal("routine should have started three times")
	}
	if resetCount.Load() != 2 {
		t.Fatal("constructor should have been called twice")
	}

	// Test conditional reset
	startCh4 := make(chan struct{})
	var startWg3 sync.WaitGroup
	startWg3.Go(func() {
		for {
			if startCount.Load() == 4 {
				close(startCh4)
				return
			}
			runtime.Gosched()
		}
	})

	existed, reset = k.ResetRoutine("test-key", func(k string, v *testData) bool {
		return v.value == "test-key-2"
	})
	if !existed || !reset {
		t.Fatal("conditional reset should have succeeded")
	}

	<-startCh4
	startWg3.Wait()
	if startCount.Load() != 4 {
		t.Fatal("routine should have started four times")
	}
	if resetCount.Load() != 3 {
		t.Fatal("constructor should have been called three times")
	}

	// Test reset all
	startCh5 := make(chan struct{})
	var startWg4 sync.WaitGroup
	startWg4.Go(func() {
		for {
			if startCount.Load() == 5 {
				close(startCh5)
				return
			}
			runtime.Gosched()
		}
	})

	resetCount2, totalCount := k.ResetAllRoutines()
	if resetCount2 != 1 || totalCount != 1 {
		t.Fatal("reset all should have reset one routine")
	}

	<-startCh5
	startWg4.Wait()
	if startCount.Load() != 5 {
		t.Fatal("routine should have started five times")
	}
	if resetCount.Load() != 4 {
		t.Fatal("constructor should have been called four times")
	}
}

// TestContextCancellation tests handling of context cancellation.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	le := logrus.NewEntry(log)

	var mu sync.Mutex
	var exitErrors []error
	exitCh := make(chan struct{})

	k := NewKeyed(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}, &testData{}
		},
		WithExitLogger[string, *testData](le),
		WithExitCb[string, *testData](func(_ string, _ Routine, _ *testData, err error) {
			mu.Lock()
			exitErrors = append(exitErrors, err)
			if len(exitErrors) == 1 {
				close(exitCh)
			}
			mu.Unlock()
		}),
	)

	k.SetContext(ctx, false)
	_, existed := k.SetKey("test-key", true)
	if existed {
		t.Fatal("key should not exist yet")
	}

	// Cancel the context
	cancel()

	// Wait for callback to be called
	<-exitCh

	mu.Lock()
	if len(exitErrors) != 1 {
		t.Fatal("should have one exit error")
	}
	if exitErrors[0] != context.Canceled {
		t.Fatalf("expected context.Canceled error, got: %v", exitErrors[0])
	}
	mu.Unlock()

	// Set a new context
	newCtx := context.Background()
	k.SetContext(newCtx, true)

	// Create a channel for the second exit
	exitCh2 := make(chan struct{})
	var exitWg sync.WaitGroup
	exitWg.Go(func() {
		for {
			mu.Lock()
			count := len(exitErrors)
			mu.Unlock()
			if count == 2 {
				close(exitCh2)
				return
			}
			runtime.Gosched()
		}
	})

	// Cancel the key
	k.RemoveKey("test-key")

	// Wait for callback to be called again
	<-exitCh2
	exitWg.Wait()

	mu.Lock()
	if len(exitErrors) != 2 {
		t.Fatal("should have two exit errors")
	}
	if exitErrors[1] != context.Canceled {
		t.Fatalf("expected context.Canceled error, got: %v", exitErrors[1])
	}
	mu.Unlock()
}
