package refcount

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/aperturerobotics/util/ccontainer"
)

// RefCount is a refcount driven object container.
// Wraps a ccontainer with a ref count mechanism.
// When there are no references, the container contents are released.
type RefCount[T comparable] struct {
	// ctx contains the root context
	// can be nil
	ctx context.Context
	// target is the target ccontainer
	target *ccontainer.CContainer[T]
	// targetErr is the destination for resolution errors
	targetErr *ccontainer.CContainer[*error]
	// resolver is the resolver function
	// returns the value and a release function
	resolver func(ctx context.Context) (T, func(), error)
	// mtx guards below fields
	mtx sync.Mutex
	// refs is the list of references.
	refs map[*Ref[T]]struct{}
	// resolveCtx is the resolution context.
	resolveCtx context.Context
	// resolveCtxCancel cancels resolveCtx
	resolveCtxCancel context.CancelFunc
	// value is the current value
	value T
	// valueErr is the current value error.
	valueErr error
	// valueRel releases the current value.
	valueRel func()
}

// Ref is a reference to a RefCount.
type Ref[T comparable] struct {
	rc  *RefCount[T]
	rel atomic.Bool
	cb  func(val T, err error)
}

// Release releases the reference.
func (k *Ref[T]) Release() {
	if k.rel.Swap(true) {
		return
	}
	k.rc.removeRef(k)
}

// NewRefCount builds a new RefCount.
// ctx, target and targetErr can be empty
func NewRefCount[T comparable](
	ctx context.Context,
	target *ccontainer.CContainer[T],
	targetErr *ccontainer.CContainer[*error],
	resolver func(ctx context.Context) (T, func(), error),
) *RefCount[T] {
	return &RefCount[T]{
		ctx:       ctx,
		target:    target,
		targetErr: targetErr,
		resolver:  resolver,
		refs:      make(map[*Ref[T]]struct{}),
	}
}

// WaitRefCountContainer waits for a RefCount container handling errors.
// targetErr can be nil
func WaitRefCountContainer[T comparable](
	ctx context.Context,
	target *ccontainer.CContainer[T],
	targetErr *ccontainer.CContainer[*error],
) (T, error) {
	var errCh chan error
	if targetErr != nil {
		errCh = make(chan error, 1)
		go func() {
			outErr, _ := targetErr.WaitValue(ctx, errCh)
			if outErr != nil && *outErr != nil {
				select {
				case errCh <- *outErr:
				default:
				}
			}
		}()
	}
	return target.WaitValue(ctx, errCh)
}

// SetContext updates the context to use for the RefCount container resolution.
// If ctx=nil the RefCount will wait until ctx != nil to start.
// This also restarts resolution, if there are any refs.
func (r *RefCount[T]) SetContext(ctx context.Context) {
	r.mtx.Lock()
	if r.ctx != ctx {
		r.ctx = ctx
		r.startResolve()
	}
	r.mtx.Unlock()
}

// AddRef adds a reference to the RefCount container.
// cb is an optional callback to call when the value changes.
func (r *RefCount[T]) AddRef(cb func(val T, err error)) *Ref[T] {
	r.mtx.Lock()
	nref := &Ref[T]{rc: r, cb: cb}
	r.refs[nref] = struct{}{}
	if len(r.refs) == 1 {
		r.startResolve()
	} else {
		var empty T
		if val := r.value; val != empty {
			nref.cb(val, nil)
		} else if err := r.valueErr; err != nil {
			nref.cb(empty, err)
		}
	}
	r.mtx.Unlock()
	return nref
}

// Wait adds a reference and waits for a value.
// Returns the value, reference, and any error.
// If err != nil, value and reference will be nil.
func (r *RefCount[T]) Wait(ctx context.Context) (T, *Ref[T], error) {
	var done atomic.Bool
	valCh := make(chan T, 1)
	errCh := make(chan error, 1)
	ref := r.AddRef(func(val T, err error) {
		if done.Swap(true) {
			return
		}
		if err != nil {
			errCh <- err
		} else {
			valCh <- val
		}
	})
	select {
	case <-ctx.Done():
		ref.Release()
		var empty T
		return empty, nil, context.Canceled
	case err := <-errCh:
		ref.Release()
		var empty T
		return empty, nil, err
	case val := <-valCh:
		return val, ref, nil
	}
}

// Access adds a reference, waits for a value, and calls the callback.
// Releases the reference once the callback has returned.
func (r *RefCount[T]) Access(ctx context.Context, cb func(T) error) error {
	val, rel, err := r.Wait(ctx)
	if err != nil {
		return err
	}
	defer rel.Release()
	return cb(val)

}

// removeRef removes a reference and shuts down if no refs remain.
func (r *RefCount[T]) removeRef(ref *Ref[T]) {
	r.mtx.Lock()
	lenBefore := len(r.refs)
	delete(r.refs, ref)
	lenAfter := len(r.refs)
	if lenAfter < lenBefore && lenAfter == 0 {
		r.shutdown()
	}
	r.mtx.Unlock()
}

// shutdown shuts down the resolver, if any.
// expects mtx is locked by caller
func (r *RefCount[T]) shutdown() {
	if r.resolveCtxCancel != nil {
		r.resolveCtxCancel()
		r.resolveCtx, r.resolveCtxCancel = nil, nil
	}
	if r.valueRel != nil {
		r.valueRel()
		r.valueRel = nil
	}
	var empty T
	r.value = empty
	if r.target != nil {
		r.target.SetValue(empty)
	}
}

// startResolve starts the resolve goroutine.
// expects caller to lock mutex.
func (r *RefCount[T]) startResolve() {
	if r.resolveCtxCancel != nil {
		r.resolveCtxCancel()
	}
	if r.ctx == nil || len(r.refs) == 0 {
		r.resolveCtxCancel = nil
		return
	}
	r.resolveCtx, r.resolveCtxCancel = context.WithCancel(r.ctx)
	go r.resolve(r.resolveCtx)
}

// resolve is the goroutine to resolve the value to the container.
func (r *RefCount[T]) resolve(ctx context.Context) {
	val, valRel, err := r.resolver(ctx)

	r.mtx.Lock()
	defer r.mtx.Unlock()

	// assert we are still the resolver
	if r.resolveCtx != ctx {
		if valRel != nil {
			valRel()
		}
		return
	}

	// store the value and/or error
	r.value, r.valueErr = val, err
	r.valueRel = valRel
	if err != nil {
		if r.targetErr != nil {
			r.targetErr.SetValue(&err)
		}
	} else {
		if r.targetErr != nil {
			r.targetErr.SetValue(nil)
		}
		if r.target != nil {
			r.target.SetValue(val)
		}
	}
	for ref := range r.refs {
		if ref.cb != nil {
			ref.cb(val, err)
		}
	}
}
