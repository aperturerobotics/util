package routine

import (
	"context"

	"github.com/sirupsen/logrus"
)

// StateRoutineContainer contains a Routine which is restarted when the input State changes.
type StateRoutineContainer[T comparable] struct {
	// rc is the routine container
	rc *RoutineContainer
	// compare compares if the two states are equivalent
	// if nil restarts the routine every time SetState is called
	compare func(t1, t2 T) bool
	// stateRoutine contains the state routine function
	stateRoutine StateRoutine[T]
	// s contains the current state
	// guarded by rc.mtx
	s T
}

// StateRoutine is a function called as a goroutine with a state parameter.
// If the state changes, ctx will be canceled and the function restarted.
// If nil is returned, exits cleanly permanently.
// If an error is returned, can still be restarted later.
type StateRoutine[T comparable] func(ctx context.Context, st T) error

// NewStateRoutineContainer constructs a new StateRoutineContainer.
//
// Note: routines won't start until SetContext and SetState is called.
// If the state is equivalent to an empty T (nil if a pointer) the routine is stopped.
// compare must compare if the two states are equivalent.
// if compare is nil restarts the routine every time SetState is called.
func NewStateRoutineContainer[T comparable](compare func(t1, t2 T) bool, opts ...Option) *StateRoutineContainer[T] {
	return &StateRoutineContainer[T]{
		rc:      NewRoutineContainer(opts...),
		compare: compare,
	}
}

// NewRoutineContainerWithLogger constructs a new RoutineContainer instance.
// Logs when a controller exits without being canceled.
//
// Note: routines won't start until SetContext is called.
func NewStateRoutineContainerWithLogger[T comparable](compare func(t1, t2 T) bool, le *logrus.Entry, opts ...Option) *StateRoutineContainer[T] {
	return &StateRoutineContainer[T]{
		rc:      NewRoutineContainerWithLogger(le, opts...),
		compare: compare,
	}
}

// GetState returns the immediate state in the StateRoutineContainer.
func (s *StateRoutineContainer[T]) GetState() T {
	s.rc.mtx.Lock()
	defer s.rc.mtx.Unlock()
	return s.s
}

// SetState sets the state in the StateRoutineContainer.
//
// Returns if the state changed and if the routine is running.
// If reset=true the existing routine was canceled or restarted.
func (s *StateRoutineContainer[T]) SetState(state T) (waitReturn <-chan struct{}, changed, reset, running bool) {
	s.rc.mtx.Lock()
	defer s.rc.mtx.Unlock()
	return s.setStateLocked(state)
}

// setStateLocked compares and updates the state when mtx is locked.
func (s *StateRoutineContainer[T]) setStateLocked(state T) (waitReturn <-chan struct{}, changed, reset, running bool) {
	if s.compare == nil {
		changed = true
	} else {
		changed = !s.compare(s.s, state)
	}
	if changed {
		s.s = state
		waitReturn, reset, running = s.updateStateRoutineLocked()
	}
	return
}

// SwapState locks the container, calls the callback, and stores the returned value.
//
// Returns the updated value and if the state changed.
// If reset=true returns a channel which closes when the previous instance has exited.
func (s *StateRoutineContainer[T]) SwapValue(cb func(val T) T) (nextState T, waitReturn <-chan struct{}, changed, reset, running bool) {
	s.rc.mtx.Lock()
	defer s.rc.mtx.Unlock()

	stateBefore := s.s
	if cb != nil {
		nextState = cb(stateBefore)
		changed = nextState != stateBefore
	} else {
		nextState = stateBefore
	}

	if changed {
		waitReturn, changed, reset, running = s.setStateLocked(nextState)
		if !changed {
			nextState = stateBefore
		}
	} else {
		running = s.rc.getRunningLocked()
	}

	return
}

// SetStateRoutine sets the routine to execute, resetting the existing, if set.
// If the specified routine is nil, shuts down the current routine.
// Returns if the current routine was stopped or overwritten.
// Returns a channel which will be closed when the previous routine exits.
// The waitReturn channel will be nil if there was no previous routine (reset=false).
// If SetContext has not been called or SetState is empty, returns false for running.
// Note: does not check if routine is equal to the current routine func (cannot compare generic funcs).
func (s *StateRoutineContainer[T]) SetStateRoutine(routine StateRoutine[T]) (waitReturn <-chan struct{}, reset, running bool) {
	s.rc.mtx.Lock()
	defer s.rc.mtx.Unlock()
	s.stateRoutine = routine
	return s.updateStateRoutineLocked()
}

// updateStateRoutineLocked updates the state or the s.stateRoutine and calls setRoutineLocked.
func (s *StateRoutineContainer[T]) updateStateRoutineLocked() (waitReturn <-chan struct{}, reset, running bool) {
	var empty T
	st, routine := s.s, s.stateRoutine
	var setRoutine Routine
	if routine != nil && st != empty {
		setRoutine = func(ctx context.Context) error {
			return routine(ctx, st)
		}
	}
	waitReturn, reset = s.rc.setRoutineLocked(setRoutine)
	running = s.rc.getRunningLocked()
	return
}

// SetContext updates the root context.
//
// nil context is valid and will shutdown the routines.
// if restart is true, errored routines will also restart.
//
// Returns if the routine was stopped or restarted.
func (s *StateRoutineContainer[T]) SetContext(ctx context.Context, restart bool) bool {
	return s.rc.SetContext(ctx, restart)
}

// ClearContext clears the context and shuts down all routines.
//
// Returns if the routine was stopped or restarted.
func (s *StateRoutineContainer[T]) ClearContext() bool {
	return s.rc.ClearContext()
}

// RestartRoutine restarts the existing routine (if set).
// Returns if the routine was restarted.
// Returns false if the context is currently nil or the routine is unset.
func (s *StateRoutineContainer[T]) RestartRoutine() bool {
	return s.rc.RestartRoutine()
}

// WaitExited waits for the routine to exit and returns the error if any.
// Note: Will NOT return after the routine is restarted normally.
// If returnIfNotRunning is set, returns nil if no routine is running.
// errCh is an optional error channel (can be nil)
func (s *StateRoutineContainer[T]) WaitExited(ctx context.Context, returnIfNotRunning bool, errCh <-chan error) error {
	return s.rc.WaitExited(ctx, returnIfNotRunning, errCh)
}
