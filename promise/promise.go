package promise

import (
	"context"
	"sync/atomic"
)

// Promise is an asynchronous result to an operation.
type Promise[T any] struct {
	// isDone is an atomic int indicating the promise has been resolved.
	isDone atomic.Bool
	// done is closed when the promise has been completed.
	done chan struct{}
	// result is the result of the promise.
	result *T
	// err is the error result of the promise.
	err error
}

// NewPromise constructs a new empty Promise.
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{done: make(chan struct{})}
}

// NewPromiseWithResult constructs a promise pre-resolved with a result.
func NewPromiseWithResult[T any](val T, err error) *Promise[T] {
	p := &Promise[T]{
		done:   make(chan struct{}),
		result: &val,
		err:    err,
	}
	close(p.done)
	p.isDone.Store(true)
	return p
}

// NewPromiseWithErr constructs a promise pre-resolved with an error.
func NewPromiseWithErr[T any](err error) *Promise[T] {
	var empty T
	return NewPromiseWithResult(empty, err)
}

// SetResult sets the result of the promise.
//
// Returns false if the result was already set.
func (p *Promise[T]) SetResult(val T, err error) bool {
	if p.isDone.Swap(true) {
		return false
	}
	p.result = &val
	p.err = err
	close(p.done)
	return true
}

// Await waits for the result to be set or for ctx to be canceled.
func (p *Promise[T]) Await(ctx context.Context) (val T, err error) {
	select {
	case <-ctx.Done():
		return val, context.Canceled
	case <-p.done:
		return *p.result, p.err
	}
}

// AwaitWithErrCh waits for the result to be set or for an error to be pushed to the channel.
func (p *Promise[T]) AwaitWithErrCh(ctx context.Context, errCh <-chan error) (val T, err error) {
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

	case <-p.done:
		return *p.result, p.err
	}
}

// AwaitWithCancelCh waits for the result to be set or for the channel to be written to and/or closed.
// Returns nil, context.Canceled if the cancelCh reads.
func (p *Promise[T]) AwaitWithCancelCh(ctx context.Context, cancelCh <-chan struct{}) (val T, err error) {
	select {
	case <-ctx.Done():
		return val, context.Canceled
	case <-cancelCh:
		return val, context.Canceled
	case <-p.done:
		return *p.result, p.err
	}
}

// _ is a type assertion
var _ PromiseLike[bool] = ((*Promise[bool])(nil))
