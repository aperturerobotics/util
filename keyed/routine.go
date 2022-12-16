package keyed

import (
	"context"
	"time"
)

// runningRoutine tracks a running routine
type runningRoutine[T comparable] struct {
	// k is the keyed instance
	k *Keyed[T]
	// key is the key for this routine
	key string

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
	data T
	// err is the error if any
	err error
	// success indicates the routine succeeded
	success bool
	// deferRemove is set if we are waiting to remove this.
	deferRemove *time.Timer
}

// newRunningRoutine constructs a new runningRoutine
func newRunningRoutine[T comparable](k *Keyed[T], key string, routine Routine, data T) *runningRoutine[T] {
	return &runningRoutine[T]{
		k:       k,
		key:     key,
		routine: routine,
		data:    data,
	}
}

// start starts or restarts the routine (if not running).
// expects k.mtx to be locked by caller
// if waitCh != nil, waits for waitCh to be closed before fully starting.
func (r *runningRoutine[T]) start(ctx context.Context, waitCh <-chan struct{}) {
	if r.success || r.routine == nil {
		return
	}
	if r.ctx != nil {
		select {
		case <-r.ctx.Done():
		default:
			// routine is still running
			return
		}
	}
	r.err = nil
	if r.ctxCancel != nil {
		r.ctxCancel()
	}
	exitedCh := make(chan struct{})
	r.exitedCh = exitedCh
	r.ctx, r.ctxCancel = context.WithCancel(ctx)
	go r.execute(r.ctx, r.ctxCancel, exitedCh, waitCh)
}

// execute executes the routine.
func (r *runningRoutine[T]) execute(
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
	} else {
		select {
		case <-ctx.Done():
			err = context.Canceled
		default:
		}
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
		r.ctxCancel, r.ctxCancel = nil, nil
		r.exitedCh = nil
		for i := len(r.k.exitedCbs) - 1; i >= 0; i-- {
			// run after unlocking mtx
			defer (r.k.exitedCbs[i])(r.key, r.routine, r.data, r.err)
		}
	}
	r.k.mtx.Unlock()
}

// remove is called when the routine is removed / canceled.
// expects r.k.mtx to be locked
func (r *runningRoutine[T]) remove() {
	if r.deferRemove != nil {
		return
	}
	removeNow := func() {
		if r.ctxCancel != nil {
			r.ctxCancel()
		}
		delete(r.k.routines, r.key)
	}
	if r.k.releaseDelay == 0 {
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
