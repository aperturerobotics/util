package promise

import (
	"context"
	"sync"

	"github.com/aperturerobotics/util/broadcast"
)

// PromiseContainer contains a Promise which can be replaced with a new Promise.
//
// The zero-value of this struct is valid.
type PromiseContainer[T any] struct {
	// mtx guards below fields
	mtx sync.Mutex
	// promise contains the current promise.
	promise PromiseLike[T]
	// replaced is broadcasted when the promise is replaced.
	replaced broadcast.Broadcast
}

// NewPromiseContainer constructs a new PromiseContainer.
func NewPromiseContainer[T any]() *PromiseContainer[T] {
	return &PromiseContainer[T]{}
}

// GetPromise returns the Promise contained in the PromiseContainer and a
// channel that is closed when the Promise is replaced.
//
// Note that promise may be nil.
func (c *PromiseContainer[T]) GetPromise() (PromiseLike[T], <-chan struct{}) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.promise, c.replaced.GetWaitCh()
}

// SetPromise updates the Promise contained in the PromiseContainer.
// Note: this does not do anything with the old promise.
func (c *PromiseContainer[T]) SetPromise(p PromiseLike[T]) {
	c.mtx.Lock()
	c.promise = p
	c.replaced.Broadcast()
	c.mtx.Unlock()
}

// SetResult sets the result of the promise.
//
// Overwrites the existing promise with a new promise.
func (p *PromiseContainer[T]) SetResult(val T, err error) bool {
	p.mtx.Lock()
	p.promise = NewPromiseWithResult(val, err)
	p.replaced.Broadcast()
	p.mtx.Unlock()
	return true
}

// Await waits for the result to be set or for ctx to be canceled.
func (p *PromiseContainer[T]) Await(ctx context.Context) (val T, err error) {
	for {
		p.mtx.Lock()
		replaceCh := p.replaced.GetWaitCh()
		promise := p.promise
		p.mtx.Unlock()
		if promise == nil {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			case <-replaceCh:
				continue
			}
		}

		val, valErr := promise.AwaitWithCancelCh(ctx, replaceCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			default:
			}
		} else {
			return val, valErr
		}
	}
}

// AwaitWithErrCh waits for the result to be set or for an error to be pushed to the channel.
func (p *PromiseContainer[T]) AwaitWithErrCh(ctx context.Context, errCh <-chan error) (val T, err error) {
	for {
		p.mtx.Lock()
		replaceCh := p.replaced.GetWaitCh()
		promise := p.promise
		p.mtx.Unlock()
		if promise == nil {
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
			case <-replaceCh:
			}
			continue
		}

		val, valErr := promise.AwaitWithCancelCh(ctx, replaceCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			default:
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
		p.mtx.Lock()
		replaceCh := p.replaced.GetWaitCh()
		promise := p.promise
		p.mtx.Unlock()
		if promise == nil {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			case <-cancelCh:
				return val, err
			case <-replaceCh:
			}
			continue
		}

		val, valErr := promise.AwaitWithCancelCh(ctx, replaceCh)
		if valErr == nil {
			return val, nil
		}
		if valErr == context.Canceled {
			select {
			case <-ctx.Done():
				return val, context.Canceled
			default:
			}
		} else {
			return val, valErr
		}
	}
}

// _ is a type assertion
var _ PromiseLike[bool] = ((*PromiseContainer[bool])(nil))
