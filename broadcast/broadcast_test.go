package broadcast

import (
	"context"
	"fmt"
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
	b.Wait(ctx, func(broadcast func()) (bool, error) {
		gotValue = currValue
		return gotValue == 9, nil
	})

	fmt.Printf("waited for value to increment: %v\n", gotValue)
	// Output: waited for value to increment: 9
}
