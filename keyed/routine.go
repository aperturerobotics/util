package keyed

import (
	"context"
	"time"

	"github.com/aperturerobotics/util/backoff"
	cbackoff "github.com/aperturerobotics/util/backoff/cbackoff"
)

// runningRoutine tracks a running routine
type runningRoutine[K comparable, V any] struct {
	// k is the keyed instance
	k *Keyed[K, V]
	// key is the key for this routine
	key K

	// fields guarded by k.mtx
	// ctx is the context
	ctx context.Context
	// ctxCancel cancels the context
	// if nil, not running
	ctxCancel context.CancelFunc
	// exitedCh is closed when the routine running with ctx exits
	// may be nil if ctx == nil
	exitedCh <-chan struct{}
	// routine is the routine callback
	routine Routine
	// data is the associated routine data
	data V
	// err is the error if any
	err error
	// success indicates the routine succeeded
	success bool
	// exited indicates the routine exited
	exited bool

	// deferRemove is set if we are waiting to remove this.
	deferRemove *time.Timer

	// retryBo is the retry backoff if retrying is enabled.
	retryBo cbackoff.BackOff
	// deferRetry is set if we are waiting to retry this.
	deferRetry *time.Timer
}

// newRunningRoutine constructs a new runningRoutine
func newRunningRoutine[K comparable, V any](
	k *Keyed[K, V],
	key K,
	routine Routine,
	data V,
	backoffFactory func(k K) cbackoff.BackOff,
) *runningRoutine[K, V] {
	var backoff cbackoff.BackOff
	if backoffFactory != nil {
		backoff = backoffFactory(key)
	}
	return &runningRoutine[K, V]{
		k:       k,
		key:     key,
		routine: routine,
		data:    data,
		retryBo: backoff,
	}
}

// start starts or restarts the routine (if not running).
// expects k.mtx to be locked by caller
// if waitCh != nil, waits for waitCh to be closed before fully starting.
// if forceRestart is set, cancels the existing routine.
func (r *runningRoutine[K, V]) start(ctx context.Context, waitCh <-chan struct{}, forceRestart bool) {
	if (!forceRestart && r.success) || r.routine == nil {
		return
	}
	if !forceRestart && r.ctx != nil && !r.exited && r.ctx.Err() == nil {
		// routine is still running
		return
	}
	if r.deferRetry != nil {
		// cancel retrying this key
		_ = r.deferRetry.Stop()
		r.deferRetry = nil
	}
	if r.ctxCancel != nil {
		r.ctxCancel()
	}
	exitedCh := make(chan struct{})
	r.err = nil
	r.success, r.exited = false, false
	r.exitedCh = exitedCh
	r.ctx, r.ctxCancel = context.WithCancel(ctx)
	go r.execute(r.ctx, r.ctxCancel, exitedCh, waitCh)
}

// execute executes the routine.
func (r *runningRoutine[K, V]) execute(
	ctx context.Context,
	cancel context.CancelFunc,
	exitedCh chan struct{},
	waitCh <-chan struct{},
) {
	var err error
	if waitCh != nil {
		select {
		case <-ctx.Done():
			err = context.Canceled
		case <-waitCh:
		}
	} else if err = ctx.Err(); err != nil {
		err = context.Canceled
	}

	if err == nil {
		err = r.routine(ctx)
	}
	cancel()
	close(exitedCh)

	r.k.mtx.Lock()
	if r.ctx == ctx {
		r.err = err
		r.success = err == nil
		r.exited = true
		r.exitedCh = nil
		if r.retryBo != nil {
			if r.deferRetry != nil {
				r.deferRetry.Stop()
				r.deferRetry = nil
			}
			if r.success {
				r.retryBo.Reset()
			} else if r.k.routines[r.key] == r {
				dur := r.retryBo.NextBackOff()
				if dur != backoff.Stop {
					r.deferRetry = time.AfterFunc(dur, func() {
						r.k.mtx.Lock()
						if r.k.ctx != nil && r.k.routines[r.key] == r && r.exited {
							r.start(r.k.ctx, r.exitedCh, true)
						}
						r.k.mtx.Unlock()
					})
				}
			}
		}
		for i := len(r.k.exitedCbs) - 1; i >= 0; i-- {
			// run after unlocking mtx
			defer (r.k.exitedCbs[i])(r.key, r.routine, r.data, r.err)
		}
	}
	r.k.mtx.Unlock()
}

// remove is called when the routine is removed / canceled.
// expects r.k.mtx to be locked
func (r *runningRoutine[K, V]) remove() {
	if r.deferRemove != nil {
		return
	}
	removeNow := func() {
		if r.ctxCancel != nil {
			r.ctxCancel()
		}
		if r.deferRetry != nil {
			// cancel retrying this key
			_ = r.deferRetry.Stop()
			r.deferRetry = nil
		}
		delete(r.k.routines, r.key)
	}
	if r.k.releaseDelay == 0 || (r.exited && !r.success) {
		removeNow()
		return
	}

	timerCb := func() {
		r.k.mtx.Lock()
		if r.k.routines[r.key] == r && r.deferRemove != nil {
			_ = r.deferRemove.Stop()
			r.deferRemove = nil
			removeNow()
		}
		r.k.mtx.Unlock()
	}
	r.deferRemove = time.AfterFunc(r.k.releaseDelay, timerCb)
}
