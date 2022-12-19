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
	promise *Promise[T]
	// replaced is broadcasted when the promise is replaced.
	replaced broadcast.Broadcast
}

// NewPromiseContainer constructs a new PromiseContainer.
func NewPromiseContainer[T any]() *PromiseContainer[T] {
	return &PromiseContainer[T]{}
}

// SetPromise updates the Promise contained in the PromiseContainer.
// Note: this does not do anything with the old promise.
func (c *PromiseContainer[T]) SetPromise(p *Promise[T]) {
	c.mtx.Lock()
	c.promise = p
	c.replaced.Broadcast()
	c.mtx.Unlock()
}

// SetResult sets the result of the promise.
//
// Returns false if the result was already set.
func (p *PromiseContainer[T]) SetResult(val T, err error) bool {
	p.mtx.Lock()
	setResult := p.promise == nil || !p.promise.isDone.Load()
	if setResult {
		p.promise = NewPromiseWithResult[T](val, err)
		p.replaced.Broadcast()
	}
	p.mtx.Unlock()
	return setResult
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

		select {
		case <-ctx.Done():
			return val, context.Canceled
		case <-replaceCh:
			continue
		case <-promise.done:
			return *promise.result, promise.err
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
			case err := <-errCh:
				return val, err
			case <-replaceCh:
				continue
			}
		}

		select {
		case <-ctx.Done():
			return val, context.Canceled
		case err := <-errCh:
			return val, err
		case <-replaceCh:
			continue
		case <-promise.done:
			return *promise.result, promise.err
		}
	}
}

// _ is a type assertion
var _ PromiseLike[bool] = ((*PromiseContainer[bool])(nil))
