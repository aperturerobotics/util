package broadcast

import (
	"context"
	"errors"
	"sync"
)

// WaitAny waits until ctx is canceled or any non-nil wait channel is closed.
//
// It is meant for watch loops that already collected wait channels while
// holding their owning locks. Nil channels are ignored. If all wait channels
// are nil, WaitAny waits for ctx cancellation. Multiple wait channels are
// joined with a cancelable fan-in so the losing waiters exit before WaitAny
// returns.
func WaitAny(ctx context.Context, waitChs ...<-chan struct{}) error {
	if ctx == nil {
		return errors.New("ctx must be set")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	waitChs = compactWaitChs(waitChs)
	switch len(waitChs) {
	case 0:
		<-ctx.Done()
		return ctx.Err()
	case 1:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-waitChs[0]:
			return nil
		}
	default:
		return waitAnyFanIn(ctx, waitChs)
	}
}

func compactWaitChs(waitChs []<-chan struct{}) []<-chan struct{} {
	var out []<-chan struct{}
	for _, ch := range waitChs {
		if ch != nil {
			out = append(out, ch)
		}
	}
	return out
}

func waitAnyFanIn(ctx context.Context, waitChs []<-chan struct{}) error {
	waitCtx, cancelWait := context.WithCancel(ctx)
	defer cancelWait()

	done := make(chan struct{})
	var closeDone sync.Once
	wake := func() {
		closeDone.Do(func() {
			close(done)
		})
	}

	for _, ch := range waitChs {
		go func() {
			select {
			case <-waitCtx.Done():
			case <-ch:
				wake()
			}
		}()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
