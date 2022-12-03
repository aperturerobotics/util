package ccall

import (
	"context"
	"sync"

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

	var mtx sync.Mutex
	var bcast broadcast.Broadcast
	var running int
	var exitErr error
	mtx.Lock()
	for _, fn := range fns {
		if fn == nil {
			continue
		}
		running++
		go func(fn CallConcurrentlyFunc) {
			err := fn(subCtx)
			mtx.Lock()
			running--
			if err != nil && (exitErr == nil || exitErr == context.Canceled) {
				exitErr = err
			}
			bcast.Broadcast()
			mtx.Unlock()
		}(fn)
	}
	mtx.Unlock()
	if running == 0 {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-bcast.GetWaitCh():
		}

		mtx.Lock()
		currRunning := running
		currExitErr := exitErr
		mtx.Unlock()
		if currRunning == 0 || currExitErr != nil {
			return currExitErr
		}
	}
}
