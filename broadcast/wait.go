package broadcast

import (
	"context"
	"errors"
)

// WaitAny waits until ctx is canceled or any non-nil wait channel is closed.
//
// It is meant for watch loops that already collected wait channels while
// holding their owning locks. Nil channels are ignored. If all wait channels
// are nil, WaitAny waits for ctx cancellation. The native implementation uses
// reflect.Select so arbitrary channel counts do not require per-channel
// goroutines. TinyGo does not implement reflect.Select, so its build uses a
// cancelable fan-in and cancels the losing waiters before returning.
func WaitAny(ctx context.Context, waitChs ...<-chan struct{}) error {
	if ctx == nil {
		return errors.New("ctx must be set")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	return waitAny(ctx, waitChs)
}
