package promise

import (
	"context"

	"github.com/aperturerobotics/util/broadcast"
)

// PromiseContainer contains a Promise which can be replaced with a new Promise.
//
// The zero-value of this struct is valid.
type PromiseContainer[T any] struct {
	// bcast is broadcasted when the promise is replaced.
	// guards below fields
	bcast broadcast.Broadcast
	// promise contains the current promise.
	promise PromiseLike[T]
}

// NewPromiseContainer constructs a new PromiseContainer.
func NewPromiseContainer[T any]() *PromiseContainer[T] {
	return &PromiseContainer[T]{}
}

// GetPromise returns the Promise contained in the PromiseContainer and a
// channel that is closed when the Promise is replaced.
//
// Note that promise may be nil.
func (c *PromiseContainer[T]) GetPromise() (prom PromiseLike[T], waitCh <-chan struct{}) {
	c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		prom, waitCh = c.promise, getWaitCh()
	})
	return
}

// SetPromise updates the Promise contained in the PromiseContainer.
// Note: this does not do anything with the old promise.
func (c *PromiseContainer[T]) SetPromise(p PromiseLike[T]) {
	c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		if c.promise != p {
			c.promise = p
			broadcast()
		}
	})
}

// SetResult sets the result of the promise.
//
// Overwrites the existing promise with a new promise.
func (p *PromiseContainer[T]) SetResult(val T, err error) bool {
	prom := NewPromiseWithResult(val, err)
	p.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		p.promise = prom
		broadcast()
	})
	return true
}

// Await waits for the result to be set or for ctx to be canceled.
func (p *PromiseContainer[T]) Await(ctx context.Context) (val T, err error) {
	for {
		var waitCh <-chan struct{}
		var prom PromiseLike[T]
		p.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			prom, waitCh = p.promise, getWaitCh()
		})
		if prom == nil {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			case <-waitCh:
				continue
			}
		}

		val, valErr := prom.AwaitWithCancelCh(ctx, waitCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			if ctx.Err() != nil {
				return val, context.Canceled
			}
		} else {
			return val, valErr
		}
	}
}

// AwaitWithErrCh waits for the result to be set or for an error to be pushed to the channel.
func (p *PromiseContainer[T]) AwaitWithErrCh(ctx context.Context, errCh <-chan error) (val T, err error) {
	for {
		var waitCh <-chan struct{}
		var prom PromiseLike[T]
		p.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			prom, waitCh = p.promise, getWaitCh()
		})
		if prom == nil {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			case err, ok := <-errCh:
				if !ok {
					// errCh was non-nil but was closed
					// treat this as context canceled
					return val, context.Canceled
				}
				return val, err
			case <-waitCh:
				continue
			}
		}

		val, valErr := prom.AwaitWithCancelCh(ctx, waitCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			if ctx.Err() != nil {
				return val, context.Canceled
			}
		} else {
			return val, valErr
		}
	}
}

// AwaitWithCancelCh waits for the result to be set or for the channel to be written to and/or closed.
//
// CancelCh could be a context.Done() channel.
//
// Will return nil, nil if the cancelCh is closed.
// Returns nil, context.Canceled if ctx is canceled.
// Otherwise waits for a value or an error to be set to the promise.
func (p *PromiseContainer[T]) AwaitWithCancelCh(ctx context.Context, cancelCh <-chan struct{}) (val T, err error) {
	for {
		var waitCh <-chan struct{}
		var prom PromiseLike[T]
		p.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			prom, waitCh = p.promise, getWaitCh()
		})
		if prom == nil {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			case <-cancelCh:
				return val, nil
			case <-waitCh:
				continue
			}
		}

		val, valErr := prom.AwaitWithCancelCh(ctx, waitCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			if ctx.Err() != nil {
				return val, context.Canceled
			}
		} else {
			return val, valErr
		}
	}
}

// _ is a type assertion
var _ PromiseLike[bool] = ((*PromiseContainer[bool])(nil))
