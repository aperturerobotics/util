package ccall

import (
	"context"

	"github.com/aperturerobotics/util/broadcast"
)

// CallConcurrentlyFunc is a function passed to CallConcurrently.
type CallConcurrentlyFunc = func(ctx context.Context) error

// CallConcurrently calls multiple functions concurrently and waits for exit or error.
func CallConcurrently(ctx context.Context, fns ...CallConcurrentlyFunc) error {
	if len(fns) == 0 {
		return nil
	}

	subCtx, subCtxCancel := context.WithCancel(ctx)
	defer subCtxCancel()
	if len(fns) == 1 {
		return fns[0](subCtx)
	}

	var bcast broadcast.Broadcast
	var running int
	var exitErr error

	callFunc := func(fn CallConcurrentlyFunc) {
		err := fn(subCtx)
		bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			running--
			if err != nil && (exitErr == nil || exitErr == context.Canceled) {
				exitErr = err
			}
			broadcast()
		})
	}

	var waitCh <-chan struct{}
	bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		waitCh = getWaitCh()
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			running++
			go callFunc(fn)
		}
	})
	if running == 0 {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-waitCh:
		}

		var currRunning int
		var currExitErr error
		bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			currRunning = running
			currExitErr = exitErr
			waitCh = getWaitCh()
		})
		if currRunning == 0 || (currExitErr != nil && currExitErr != context.Canceled) {
			return currExitErr
		}
	}
}
