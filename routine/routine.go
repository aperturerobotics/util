package routine

import (
	"context"
	"time"

	"github.com/aperturerobotics/util/backoff"
	"github.com/aperturerobotics/util/broadcast"
	cbackoff "github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

// Routine is a function called as a goroutine.
// If nil is returned, exits cleanly permanently.
// If an error is returned, can be restarted later.
type Routine func(ctx context.Context) error

// RoutineContainer contains a Routine.
type RoutineContainer struct {
	// exitedCbs is the set of exited callbacks.
	exitedCbs []func(err error)
	// bcast guards below fields
	bcast broadcast.Broadcast
	// ctx is the current root context
	ctx context.Context
	// routine is the current running routine, if any
	routine *runningRoutine
	// retryBo is the retry backoff if retrying is enabled.
	retryBo cbackoff.BackOff
}

// NewRoutineContainer constructs a new RoutineContainer.
// Note: routines won't start until SetContext is called.
func NewRoutineContainer(opts ...Option) *RoutineContainer {
	c := &RoutineContainer{}
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyToRoutineContainer(c)
		}
	}
	return c
}

// NewRoutineContainerWithLogger constructs a new RoutineContainer instance.
// Logs when a controller exits without being canceled.
//
// Note: routines won't start until SetContext is called.
func NewRoutineContainerWithLogger(le *logrus.Entry, opts ...Option) *RoutineContainer {
	return NewRoutineContainer(append([]Option{WithExitLogger(le)}, opts...)...)
}

// WaitExited waits for the routine to exit and returns the error if any.
// Note: Will NOT return after the routine is restarted normally.
// If returnIfNotRunning is set, returns nil if no routine is running.
// If returnIfNotRunning is not set, waits until a routine has started & exited.
// errCh is an optional error channel (can be nil)
func (k *RoutineContainer) WaitExited(ctx context.Context, returnIfNotRunning bool, errCh <-chan error) error {
	for {
		var exited bool
		var exitedErr error
		var waitCh <-chan struct{}

		k.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			if k.ctx != nil && k.ctx.Err() != nil {
				k.ctx = nil
			}
			if k.routine != nil && k.ctx != nil {
				exited = k.routine.exited || k.routine.success
				if exited {
					exitedErr = k.routine.err
				}
			} else if returnIfNotRunning {
				exited = true
			}
			waitCh = getWaitCh()
		})

		if exited {
			return exitedErr
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case err, ok := <-errCh:
			if !ok {
				// errCh was closed
				return context.Canceled
			}
			return err
		case <-waitCh:
		}
	}
}

// SetContext updates the root context.
//
// nil context is valid and will shutdown the routines.
// if restart is true, errored routines will also restart.
//
// Returns if the routine was stopped or restarted.
func (k *RoutineContainer) SetContext(ctx context.Context, restart bool) bool {
	var changed bool
	k.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		sameCtx := k.ctx == ctx
		if sameCtx && !restart {
			return
		}

		k.ctx = ctx
		rr := k.routine
		if rr == nil || (sameCtx && rr.err == nil) {
			return
		}

		rr.stop()
		if rr.err == nil || restart {
			if ctx != nil {
				rr.start(ctx, rr.exitedCh, false)
			}
		}

		changed = true
		broadcast()
	})
	return changed
}

// ClearContext clears the context and shuts down all routines.
//
// Returns if the routine was stopped or restarted.
func (k *RoutineContainer) ClearContext() bool {
	return k.SetContext(nil, false)
}

// getRunningLocked returns if the routine is running.
func (k *RoutineContainer) getRunningLocked() bool {
	return k.ctx != nil && k.ctx.Err() == nil && k.routine != nil && !k.routine.exited
}

// SetRoutine sets the routine to execute, resetting the existing, if set.
// If the specified routine is nil, shuts down the current routine.
// Returns if the current routine was stopped or overwritten.
// Returns a channel which will be closed when the previous routine exits.
// The waitReturn channel will be nil if there was no previous routine (reset=false).
func (k *RoutineContainer) SetRoutine(routine Routine) (waitReturn <-chan struct{}, reset bool) {
	k.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		waitReturn, reset = k.setRoutineLocked(routine, broadcast)
	})
	return
}

// setRoutineLocked updates the Routine while bcast is locked.
// Returns if the current routine was stopped or overwritten.
// Returns a channel to wait for in order to wait for the previous routine to exit.
func (k *RoutineContainer) setRoutineLocked(routine Routine, broadcast func()) (<-chan struct{}, bool) {
	if k.ctx != nil && k.ctx.Err() != nil {
		k.ctx = nil
	}

	var prevExitedCh <-chan struct{}
	prevRoutine := k.routine
	var wasReset bool
	if prevRoutine != nil {
		wasReset = k.ctx != nil && !prevRoutine.exited
		prevExitedCh = prevRoutine.exitedCh
		if prevRoutine.ctxCancel != nil {
			prevRoutine.ctxCancel()
			prevRoutine.ctxCancel = nil
		}
		k.routine = nil
	}

	if routine != nil {
		r := newRunningRoutine(k, routine)
		k.routine = r
		if k.ctx != nil {
			k.routine.start(k.ctx, prevExitedCh, false)
		}
		broadcast()
	} else if wasReset {
		broadcast()
	}

	return prevExitedCh, wasReset
}

// RestartRoutine restarts the existing routine (if set).
// Returns if the routine was restarted.
// Returns false if the context is currently nil or the routine is unset.
func (k *RoutineContainer) RestartRoutine() bool {
	var restarted bool
	k.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		restarted = k.restartRoutineLocked(false, broadcast)
	})
	return restarted
}

// restartRoutineLocked restarts the running routine (if set) while locked.
func (k *RoutineContainer) restartRoutineLocked(onlyIfExited bool, broadcast func()) bool {
	if k.ctx != nil && k.ctx.Err() != nil {
		k.ctx = nil
	}

	r := k.routine
	if r == nil {
		return false
	}
	if onlyIfExited && !r.exited {
		return false
	}

	if r.ctxCancel != nil {
		r.ctxCancel()
		r.ctxCancel = nil
	}
	if k.ctx == nil {
		return false
	}

	prevExitedCh := r.exitedCh
	r.exitedCh = nil
	r.start(k.ctx, prevExitedCh, true)
	broadcast()
	return true
}

// runningRoutine tracks a running routine
type runningRoutine struct {
	// r is the routine instance
	r *RoutineContainer

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
	// err is the error if any
	err error
	// success indicates the routine succeeded
	success bool
	// exited indicates the routine exited
	exited bool

	// deferRetry is set if we are waiting to retry this.
	deferRetry *time.Timer
}

// newRunningRoutine constructs a new runningRoutine
func newRunningRoutine(
	r *RoutineContainer,
	routine Routine,
) *runningRoutine {
	return &runningRoutine{
		r:       r,
		routine: routine,
	}
}

// start starts or restarts the routine (if not running).
// expects k.mtx to be locked by caller
// if waitCh != nil, waits for waitCh to be closed before fully starting.
// if forceRestart is set, cancels the existing routine.
func (r *runningRoutine) start(ctx context.Context, waitCh <-chan struct{}, forceRestart bool) {
	if (!forceRestart && r.success) || r.routine == nil {
		return
	}
	if !forceRestart && r.ctx != nil && !r.exited && r.ctx.Err() == nil {
		// routine is still running
		return
	}
	r.stop()
	exitedCh := make(chan struct{})
	r.err = nil
	r.success, r.exited = false, false
	r.exitedCh = exitedCh
	r.ctx, r.ctxCancel = context.WithCancel(ctx)
	go r.execute(r.ctx, r.ctxCancel, exitedCh, waitCh)
}

// execute executes the routine.
func (r *runningRoutine) execute(
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
	} else if ctx.Err() != nil {
		err = context.Canceled
	}

	if err == nil {
		err = r.routine(ctx)
	}
	cancel()
	close(exitedCh)

	r.r.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		if r.ctx == ctx {
			r.err = err
			r.success = err == nil
			r.exited = true
			r.exitedCh = nil
			if r.r.retryBo != nil {
				if r.deferRetry != nil {
					r.deferRetry.Stop()
					r.deferRetry = nil
				}
				if r.success {
					r.r.retryBo.Reset()
				} else if r.r.routine == r {
					dur := r.r.retryBo.NextBackOff()
					if dur != backoff.Stop {
						r.deferRetry = time.AfterFunc(dur, func() {
							r.r.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
								if r.r.ctx != nil && r.r.routine == r && r.exited {
									r.start(r.r.ctx, r.exitedCh, true)
								}
								broadcast()
							})
						})
					}
				}
			}
			for i := len(r.r.exitedCbs) - 1; i >= 0; i-- {
				// run after unlocking bcast
				defer r.r.exitedCbs[i](err)
			}
			broadcast()
		}
	})
}

// stop is called when the routine is removed / canceled.
// expects r.r.mtx to be locked
func (r *runningRoutine) stop() {
	r.ctx = nil
	if r.ctxCancel != nil {
		r.ctxCancel()
		r.ctxCancel = nil
	}
	if r.deferRetry != nil {
		_ = r.deferRetry.Stop()
		r.deferRetry = nil
	}
}
