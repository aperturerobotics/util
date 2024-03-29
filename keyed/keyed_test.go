package keyed

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aperturerobotics/util/backoff"
	"github.com/sirupsen/logrus"
)

// testData contains some test metadata.
type testData struct{}

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
	for i := 0; i < nsend; i++ {
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
	k := NewKeyed(
		func(key string) (Routine, *testData) {
			return func(ctx context.Context) error {
				called.Store(true)
				<-ctx.Done()
				canceled.Store(true)
				return nil
			}, &testData{}
		},
		WithExitLogger[string, *testData](le),
		WithReleaseDelay[string, *testData](time.Millisecond*180),
	)

	// start execution
	k.SetContext(ctx, false)

	k.SetKey("test", true)
	<-time.After(time.Millisecond * 50)
	if !called.Load() || canceled.Load() {
		t.Fail()
	}
	_ = k.RemoveKey("test")
	<-time.After(time.Millisecond * 100)
	if !called.Load() || canceled.Load() {
		t.Fail()
	}
	<-time.After(time.Millisecond * 200)
	if !called.Load() || !canceled.Load() {
		t.Fail()
	}

	canceled.Store(false)
	called.Store(false)

	k.SetKey("test", false)
	<-time.After(time.Millisecond * 50)
	if !called.Load() || canceled.Load() {
		t.Fail()
	}
	_ = k.RemoveKey("test")
	<-time.After(time.Millisecond * 100)
	if !called.Load() || canceled.Load() {
		t.Fail()
	}
	k.SetKey("test", false)
	<-time.After(time.Millisecond * 200)
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
