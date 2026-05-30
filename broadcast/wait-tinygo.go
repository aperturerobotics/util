//go:build tinygo

package broadcast

import (
	"context"
	"sync"
)

func waitAny(ctx context.Context, waitChs []<-chan struct{}) error {
	waitCtx, cancelWait := context.WithCancel(ctx)
	defer cancelWait()

	done := make(chan struct{})
	var closeDone sync.Once
	wake := func() {
		closeDone.Do(func() {
			close(done)
		})
	}

	var hasWaitCh bool
	for _, ch := range waitChs {
		if ch == nil {
			continue
		}
		hasWaitCh = true
		ch := ch
		go func() {
			select {
			case <-waitCtx.Done():
			case <-ch:
				wake()
			}
		}()
	}

	if !hasWaitCh {
		<-ctx.Done()
		return ctx.Err()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
