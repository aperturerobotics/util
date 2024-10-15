package promise

import (
	"context"
	"sync"
)

// Once contains a function that is called concurrently once.
//
// The result is returned as a promise.
// If the function returns no error, the result is stored and memoized.
//
// Otherwise, future calls to the function will try again.
type Once[T comparable] struct {
	cb   func(ctx context.Context) (T, error)
	mtx  sync.Mutex
	prom *Promise[T]
}

// NewOnce constructs a new Once caller.
func NewOnce[T comparable](cb func(ctx context.Context) (T, error)) *Once[T] {
	return &Once[T]{cb: cb}
}

// Start attempts to start resolution returning the promise.

// Resolve attempts to resolve the value using the ctx.
func (o *Once[T]) Resolve(ctx context.Context) (T, error) {
	for {
		var empty T
		if err := ctx.Err(); err != nil {
			return empty, context.Canceled
		}

		o.mtx.Lock()
		prom := o.prom

		// start if not running
		if prom == nil {
			prom = NewPromise[T]()
			o.prom = prom

			go func() {
				result, err := o.cb(ctx)
				if err != nil {
					o.mtx.Lock()
					if o.prom == prom {
						o.prom = nil
					}
					o.mtx.Unlock()

					if ctx.Err() != nil {
						prom.SetResult(empty, context.Canceled)
					} else {
						prom.SetResult(empty, err)
					}
				} else {
					prom.SetResult(result, err)
				}
			}()
		}
		o.mtx.Unlock()

		// await result
		res, err := prom.Await(ctx)
		if err == context.Canceled {
			continue
		}
		return res, err
	}
}
