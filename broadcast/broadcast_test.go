package broadcast

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func ExampleBroadcast() {
	// b guards currValue
	var b Broadcast
	var currValue int

	go func() {
		// 0 to 9 inclusive
		for i := range 10 {
			<-time.After(time.Millisecond * 20)
			locked := b.Lock()
			currValue = i
			locked.Broadcast()
			locked.Unlock()
		}
	}()

	var waitCh <-chan struct{}
	var gotValue int
	for {
		locked := b.Lock()
		gotValue = currValue
		if gotValue != 9 {
			waitCh = locked.WaitCh()
		}
		locked.Unlock()

		// last value
		if gotValue == 9 {
			// success
			break
		}

		// otherwise keep waiting
		<-waitCh
	}

	fmt.Printf("waited for value to increment: %v\n", gotValue)
	// Output: waited for value to increment: 9
}

func ExampleBroadcast_Wait() {
	// b guards currValue
	var b Broadcast
	var currValue int

	go func() {
		// 0 to 9 inclusive
		for i := range 10 {
			<-time.After(time.Millisecond * 20)
			b.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
				currValue = i
				broadcast()
			})
		}
	}()

	ctx := context.Background()
	var gotValue int
	err := b.Wait(ctx, func(broadcast func(), getWaitCh func() <-chan struct{}) (bool, error) {
		gotValue = currValue
		return gotValue == 9, nil
	})
	if err != nil {
		fmt.Printf("failed to wait for value: %v", err.Error())
		return
	}

	fmt.Printf("waited for value to increment: %v\n", gotValue)
	// Output: waited for value to increment: 9
}

func TestBroadcastCallbackIdempotent(t *testing.T) {
	var b Broadcast
	b.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		waitCh := getWaitCh()

		broadcast()
		broadcast()

		select {
		case <-waitCh:
		default:
			t.Fatal("expected wait channel to close")
		}
	})
}

func TestBroadcastWaitChannelCloseIdempotent(t *testing.T) {
	ch := newBroadcastWaitCh()

	ch.close()
	ch.close()
}

func TestBroadcastLockedAPI(t *testing.T) {
	var b Broadcast

	locked := b.Lock()
	waitCh := locked.WaitCh()
	locked.Broadcast()
	locked.Unlock()

	select {
	case <-waitCh:
	default:
		t.Fatal("expected wait channel to close")
	}

	locked, ok := b.TryLock()
	if !ok {
		t.Fatal("expected TryLock after Unlock to succeed")
	}
	locked.Unlock()
}

func TestBroadcastTryLockBusy(t *testing.T) {
	var b Broadcast

	locked := b.Lock()
	if _, ok := b.TryLock(); ok {
		t.Fatal("expected TryLock to fail while locked")
	}
	locked.Unlock()
}

func TestBroadcastLockDoesNotAllocate(t *testing.T) {
	var bc Broadcast

	allocs := testing.AllocsPerRun(100, func() {
		locked := bc.Lock()
		locked.Broadcast()
		locked.Unlock()
	})
	if allocs != 0 {
		t.Fatalf("expected Lock/Broadcast/Unlock to avoid allocations, got %v", allocs)
	}
}

func BenchmarkBroadcastHoldLock(b *testing.B) {
	var bc Broadcast
	b.ReportAllocs()
	for b.Loop() {
		bc.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			broadcast()
		})
	}
}

func BenchmarkBroadcastLock(b *testing.B) {
	var bc Broadcast
	b.ReportAllocs()
	for b.Loop() {
		locked := bc.Lock()
		locked.Broadcast()
		locked.Unlock()
	}
}
