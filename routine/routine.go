package routine

import (
	"context"
	"sync"

	"github.com/aperturerobotics/util/broadcast"
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
	// mtx guards below fields
	mtx sync.Mutex
	// ctx is the current root context
	ctx context.Context
	// routine is the current running routine, if any
	routine *runningRoutine
	// bcast is broadcasted when the routine changes
	bcast broadcast.Broadcast
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
func NewRoutineContainerWithLogger(le *logrus.Entry) *RoutineContainer {
	return NewRoutineContainer(WithExitLogger(le))
}

// WaitExited waits for the routine to exit and returns the error if any.
// Note: Will NOT return after the routine is restarted normally.
// If returnIfNotRunning is set, returns nil if no routine is running.
// errCh is an optional error channel (can be nil)
func (k *RoutineContainer) WaitExited(ctx context.Context, returnIfNotRunning bool, errCh <-chan error) error {
	for {
		k.mtx.Lock()
		var exited bool
		var exitedErr error
		if k.routine != nil {
			exited = k.routine.exited || k.routine.success
			if exited {
				exitedErr = k.routine.err
			}
		} else {
			if returnIfNotRunning {
				exited = true
			}
		}
		waitCh := k.bcast.GetWaitCh()
		k.mtx.Unlock()

		if exited {
			return exitedErr
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case err := <-errCh:
			return err
		case <-waitCh:
		}
	}
}

// SetContext updates the root context.
// If ctx == nil, stops the routine.
// if restart is true, errored routines will also restart.
// Returns if the routine was stopped or restarted.
func (k *RoutineContainer) SetContext(ctx context.Context, restart bool) bool {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	sameCtx := k.ctx == ctx
	if sameCtx && !restart {
		return false
	}

	k.ctx = ctx
	rr := k.routine
	if rr == nil || (sameCtx && rr.err == nil) {
		return false
	}
	rr.ctx = nil
	if rr.ctxCancel != nil {
		rr.ctxCancel()
		rr.ctxCancel = nil
	}
	if rr.err == nil || restart {
		if ctx != nil {
			rr.start(ctx, rr.exitedCh, false)
		}
	}
	k.bcast.Broadcast()
	return true
}

// SetRoutine sets the routine to execute, resetting the existing, if set.
// If the specified routine is nil, shuts down the current routine.
// If routine = nil, waits to return until the existing routine fully shuts down.
// Otherwise, returns right away without blocking.
// Returns if the current routine was stopped or overwritten.
func (k *RoutineContainer) SetRoutine(routine Routine) bool {
	var waitReturn <-chan struct{}
	defer func() {
		if waitReturn != nil {
			<-waitReturn
		}
	}()

	k.mtx.Lock()
	defer k.mtx.Unlock()

	if k.ctx != nil {
		select {
		case <-k.ctx.Done():
			k.ctx = nil
		default:
		}
	}

	var prevExitedCh <-chan struct{}
	prevRoutine := k.routine
	wasReset := prevRoutine != nil
	if wasReset {
		prevExitedCh = prevRoutine.exitedCh
		if prevRoutine.ctxCancel != nil {
			prevRoutine.ctxCancel()
			prevRoutine.ctxCancel = nil
		}
		k.routine = nil
	}

	k.bcast.Broadcast()
	if routine == nil {
		return wasReset
	}

	r := newRunningRoutine(k, routine)
	k.routine = r
	if k.ctx != nil {
		k.routine.start(k.ctx, prevExitedCh, false)
	} else {
		waitReturn = prevExitedCh
	}
	return wasReset
}

// RestartRoutine restarts the existing routine (if set).
// Returns if the routine was restarted.
// Returns false if the context is currently nil or the routine is unset.
func (k *RoutineContainer) RestartRoutine() bool {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	if k.ctx != nil {
		select {
		case <-k.ctx.Done():
			k.ctx = nil
		default:
		}
	}

	r := k.routine
	if r == nil {
		return false
	}

	if r.ctxCancel != nil {
		r.ctxCancel()
		r.ctxCancel = nil
	}
	k.bcast.Broadcast()
	if k.ctx == nil {
		return false
	}

	prevExitedCh := r.exitedCh
	r.exitedCh = nil
	r.start(k.ctx, prevExitedCh, true)
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
}

// newRunningRoutine constructs a new runningRoutine
func newRunningRoutine(r *RoutineContainer, routine Routine) *runningRoutine {
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
	if !forceRestart && r.ctx != nil && !r.exited {
		select {
		case <-r.ctx.Done():
		default:
			// routine is still running
			return
		}
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

	r.r.mtx.Lock()
	if r.ctx == ctx {
		r.err = err
		r.success = err == nil
		r.exited = true
		r.exitedCh = nil
		for i := len(r.r.exitedCbs) - 1; i >= 0; i-- {
			// run after unlocking mtx
			defer (r.r.exitedCbs[i])(r.err)
		}
		r.r.bcast.Broadcast()
	}
	r.r.mtx.Unlock()
}
