package refcount

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/aperturerobotics/util/broadcast"
	"github.com/aperturerobotics/util/ccontainer"
	"github.com/aperturerobotics/util/promise"
)

// RefCountResolver resolves a value within a RefCount container.
//
// Accepts a released callback which can be called when the returned value is invalid.
// Returns a release function which will be called when the returned value is no longer needed.
type RefCountResolver[T comparable] func(ctx context.Context, released func()) (T, func(), error)

// RefCount is a refcount driven object container.
// Wraps a ccontainer with a ref count mechanism.
// When there are no references, the container contents are released.
type RefCount[T comparable] struct {
	// ctx contains the root context
	// can be nil
	ctx context.Context
	// keepUnref sets if the value should be kept if there are zero references.
	keepUnref bool
	// target is the target ccontainer
	target *ccontainer.CContainer[T]
	// targetErr is the destination for resolution errors
	targetErr *ccontainer.CContainer[*error]
	// resolver is the resolver function
	// returns the value and a release function
	// call the released callback if the value is no longer valid.
	resolver RefCountResolver[T]
	// mtx guards below fields
	mtx sync.Mutex
	// refs is the list of references.
	refs map[*Ref[T]]struct{}
	// resolveCtx is the resolution context.
	resolveCtx context.Context
	// resolveCtxCancel cancels resolveCtx
	resolveCtxCancel context.CancelFunc
	// nonce is incremented when starting/stopping the resolver
	nonce uint32
	// waitCh is a channel to wait before starting next resolve
	// may be nil
	waitCh chan struct{}
	// resolved indicates the value is set
	resolved bool
	// value is the current value
	value T
	// valueErr is the current value error.
	valueErr error
	// valueRel releases the current value.
	valueRel func()
}

// RefLike is an interface implemented by Ref.
type RefLike interface {
	// Release releases the reference.
	Release()
}

// Ref is a reference to a RefCount.
type Ref[T comparable] struct {
	rc  *RefCount[T]
	rel atomic.Bool
	cb  func(resolved bool, val T, err error)
}

// Release releases the reference.
func (k *Ref[T]) Release() {
	if k.rel.Swap(true) {
		return
	}
	k.rc.removeRef(k)
}

// NewRefCount builds a new RefCount.
//
// ctx, target and targetErr can be empty
//
// keepUnref sets if the value should be kept if there are zero references.
// resolver is the resolver function
// returns the value and a release function
// call the released callback if the value is no longer valid.
func NewRefCount[T comparable](
	ctx context.Context,
	keepUnref bool,
	target *ccontainer.CContainer[T],
	targetErr *ccontainer.CContainer[*error],
	resolver RefCountResolver[T],
) *RefCount[T] {
	return &RefCount[T]{
		ctx:       ctx,
		keepUnref: keepUnref,
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
// Returns if the context was updated.
func (r *RefCount[T]) SetContext(ctx context.Context) bool {
	var updated bool
	r.mtx.Lock()
	if r.ctx != ctx {
		r.ctx = ctx
		r.startResolveLocked()
		updated = true
	}
	r.mtx.Unlock()
	return updated
}

// ClearContext clears the context and shuts down all routines.
func (r *RefCount[T]) ClearContext() {
	_ = r.SetContext(nil)
}

// AddRef adds a reference to the RefCount container.
// cb is an optional callback to call when the value changes.
// the callback will be called with an empty value when the value becomes empty.
func (r *RefCount[T]) AddRef(cb func(resolved bool, val T, err error)) *Ref[T] {
	r.mtx.Lock()
	nref := &Ref[T]{rc: r, cb: cb}
	r.refs[nref] = struct{}{}
	if len(r.refs) == 1 && !r.resolved {
		r.startResolveLocked()
	} else if r.resolved && nref.cb != nil {
		nref.cb(true, r.value, r.valueErr)
	}
	r.mtx.Unlock()
	return nref
}

// AddRefPromise adds a reference and returns a promise with the value.
func (r *RefCount[T]) AddRefPromise() (promise.PromiseLike[T], *Ref[T]) {
	promCtr := promise.NewPromiseContainer[T]()
	ref := r.AddRef(func(resolved bool, val T, err error) {
		if !resolved {
			promCtr.SetPromise(nil)
		} else {
			promCtr.SetResult(val, err)
		}
	})
	return promCtr, ref
}

// Wait adds a reference and waits for a value.
// Returns the value, reference, and any error.
// If err != nil, value and reference will be nil.
func (r *RefCount[T]) Wait(ctx context.Context) (T, *Ref[T], error) {
	prom, ref := r.AddRefPromise()
	val, err := prom.Await(ctx)
	if err != nil {
		ref.Release()
		return val, nil, err
	}
	return val, ref, nil
}

// WaitWithReleased adds a reference, waits for a value, returns the value and a release function.
// Calls the released callback (if set) when the value or reference is released.
// Note: it's very unlikely, but still possible, that released will be called before the promise resolves.
// Note: released will always be called from a new goroutine.
func (r *RefCount[T]) WaitWithReleased(ctx context.Context, released func()) (promise.PromiseLike[T], *Ref[T]) {
	prom := promise.NewPromise[T]()
	// fields guarded by r.mtx
	var currResolved bool
	var currNonce uint32
	var callReleasedOnce sync.Once
	var ref *Ref[T]
	ref = r.AddRef(func(resolved bool, val T, err error) {
		// note: r.mtx is held while calling this function.
		// check if state is different, if we returned already.
		if currResolved {
			if !resolved || r.nonce != currNonce {
				callReleasedOnce.Do(func() {
					go func() {
						ref.Release()
						if released != nil {
							released()
						}
					}()
				})
			}
			return
		}
		if resolved || err != nil {
			currResolved = true
			currNonce = r.nonce
			prom.SetResult(val, err)
		}
	})
	return prom, ref
}

// Resolve adds a reference and waits for a value.
// Returns the value, release function, and any error.
// If err != nil, value and reference will be nil.
func (r *RefCount[T]) Resolve(ctx context.Context) (T, func(), error) {
	val, ref, err := r.Wait(ctx)
	if err != nil {
		return val, nil, err
	}
	return val, ref.Release, nil
}

// ResolveWithReleased adds a reference, waits for a value, returns the value and a release function.
// Calls the released callback (if set) when the value or reference is released.
// Note: it's very unlikely, but still possible, that released will be called before the promise resolves.
// Note: released will always be called from a new goroutine.
// Note: this matches the signature of the refcount resolver function.
func (r *RefCount[T]) ResolveWithReleased(ctx context.Context, released func()) (T, func(), error) {
	prom, ref := r.WaitWithReleased(ctx, released)
	val, err := prom.Await(ctx)
	if err != nil {
		ref.Release()
		return val, nil, err
	}
	return val, ref.Release, nil
}

// Access adds a reference, waits for a value, and calls the callback.
// Releases the reference once the callback has returned.
// The context will be canceled if the value is removed / changed.
// Return context.Canceled if the context is canceled.
// The callback may be restarted if the context is canceled and a new value is resolved.
// ctx and cb cannot be nil
func (r *RefCount[T]) Access(ctx context.Context, cb func(ctx context.Context, val T) error) error {
	var bcast broadcast.Broadcast
	var currVal T
	var currErr error
	var currResolved bool
	var currNonce uint32
	var currComplete bool

	ref := r.AddRef(func(nowResolved bool, nowVal T, nowErr error) {
		bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			if nowResolved != currResolved || nowVal != currVal || nowErr != currErr {
				currVal = nowVal
				currErr = nowErr
				currResolved = nowResolved
				currNonce++
				broadcast()
			}
		})
	})
	defer ref.Release()

	for {
		// get the current state
		var val T
		var err error
		var resolved bool
		var nonce uint32
		var complete bool
		var waitCh <-chan struct{}
		bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			// mark the nonce has increased to ensure we release the correct value later.
			currNonce++
			// snapshot current state
			val, err, resolved, nonce, complete = currVal, currErr, currResolved, currNonce, currComplete
			waitCh = getWaitCh()
		})
		if err != nil || complete {
			return err
		}

		// if we have a value currently, call the callback.
		if resolved {
			cbCtx, cbCancel := context.WithCancel(ctx)

			// start a goroutine to wait until waitCh closes and cancel the ctx.
			go func() {
				select {
				case <-ctx.Done():
				case <-cbCtx.Done():
				case <-waitCh:
					cbCancel()
				}
			}()

			cbErr := func() error {
				defer cbCancel()
				return cb(cbCtx, val)
			}()

			// stop here if the context is canceled
			if ctx.Err() != nil {
				return context.Canceled
			}

			// return now if the nonce is the same (nothing changed)
			var sameNonce bool
			bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
				sameNonce = currNonce == nonce
			})
			if sameNonce {
				return cbErr
			}
		}

		// wait for something to change
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-waitCh:
		}
	}
}

// removeRef removes a reference and shuts down if no refs remain.
func (r *RefCount[T]) removeRef(ref *Ref[T]) {
	r.mtx.Lock()
	lenBefore := len(r.refs)
	delete(r.refs, ref)
	lenAfter := len(r.refs)
	if lenAfter < lenBefore && lenAfter == 0 {
		if !r.keepUnref || !r.resolved || r.valueErr != nil {
			r.shutdown()
		}
	}
	r.mtx.Unlock()
}

// shutdown shuts down the resolver and clears state.
// expects mtx is locked by caller
func (r *RefCount[T]) shutdown() {
	r.nonce++
	r.clearResolvedState()
}

// clearResolvedState clears the resolved state.
// expects mtx is locked by caller
func (r *RefCount[T]) clearResolvedState() {
	if r.resolved {
		r.resolved = false
		if r.valueErr != nil {
			r.valueErr = nil
			if r.targetErr != nil {
				r.targetErr.SetValue(nil)
			}
		}
		var empty T
		if r.value != empty {
			r.value = empty
			if r.target != nil {
				r.target.SetValue(empty)
			}
		}
		r.callRefCbsLocked(false, empty, nil)
	}
	if r.resolveCtxCancel != nil {
		r.resolveCtxCancel()
		r.resolveCtx, r.resolveCtxCancel = nil, nil
	}
	if r.valueRel != nil {
		r.valueRel()
		r.valueRel = nil
	}
}

// startResolveLocked starts the resolve goroutine.
// expects caller to lock mutex.
func (r *RefCount[T]) startResolveLocked() {
	r.shutdown()
	if r.ctx == nil || len(r.refs) == 0 {
		return
	}
	waitCh := r.waitCh
	doneCh := make(chan struct{})
	r.waitCh = doneCh
	r.resolveCtx, r.resolveCtxCancel = context.WithCancel(r.ctx)
	nonce := r.nonce
	go r.resolve(r.resolveCtx, waitCh, doneCh, nonce)
}

// resolve is the goroutine to resolve the value to the container.
func (r *RefCount[T]) resolve(ctx context.Context, waitCh, doneCh chan struct{}, nonce uint32) {
	defer close(doneCh)

	if waitCh != nil {
		select {
		case <-ctx.Done():
			return
		case <-waitCh:
		}
	}

	released := func() {
		resolveAfterRelease := func(lock bool) {
			if lock {
				r.mtx.Lock()
			}
			defer r.mtx.Unlock()
			if r.nonce == nonce {
				// calls shutdown internally
				r.startResolveLocked()
			}
		}

		if r.mtx.TryLock() {
			resolveAfterRelease(false)
		} else {
			go resolveAfterRelease(true)
		}
	}

	val, valRel, err := r.resolver(ctx, released)

	r.mtx.Lock()
	defer r.mtx.Unlock()

	// assert we are still the resolver
	if r.nonce != nonce {
		if valRel != nil {
			defer valRel()
		}
		return
	}

	// store the value and/or error
	r.resolved = true
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
	r.callRefCbsLocked(true, val, err)
}

// callRefCbsLocked calls the reference callbacks.
func (r *RefCount[T]) callRefCbsLocked(resolved bool, val T, err error) {
	for ref := range r.refs {
		if ref.cb != nil {
			ref.cb(resolved, val, err)
		}
	}
}

// _ is a type assertion
var _ RefLike = (*Ref[*struct{}])(nil)
