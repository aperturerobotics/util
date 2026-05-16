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
			b.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
				currValue = i
				broadcast()
			})
		}
	}()

	var waitCh <-chan struct{}
	var gotValue int
	for {
		b.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			gotValue = currValue
			waitCh = getWaitCh()
		})

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
