package routine

import (
	"context"

	proto "github.com/aperturerobotics/protobuf-go-lite"
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
	// guarded by rc's transition lock
	stateRoutine StateRoutine[T]
	// s contains the current state
	// guarded by rc's transition lock
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

// NewStateRoutineContainerWithLogger constructs a StateRoutineContainer that
// logs when the routine exits without being canceled.
//
// Note: routines won't start until SetContext and SetState are called.
func NewStateRoutineContainerWithLogger[T comparable](compare func(t1, t2 T) bool, le *logrus.Entry, opts ...Option) *StateRoutineContainer[T] {
	return &StateRoutineContainer[T]{
		rc:      NewRoutineContainerWithLogger(le, opts...),
		compare: compare,
	}
}

// NewStateRoutineContainerVT constructs a StateRoutineContainer that compares
// states with EqualVT.
//
// Note: routines won't start until SetContext and SetState are called.
// If the state is equivalent to an empty T (nil if a pointer) the routine is stopped.
func NewStateRoutineContainerVT[T proto.EqualVT[T]](opts ...Option) *StateRoutineContainer[T] {
	return &StateRoutineContainer[T]{
		rc:      NewRoutineContainer(opts...),
		compare: proto.CompareEqualVT[T](),
	}
}

// NewStateRoutineContainerWithLoggerVT constructs a StateRoutineContainer that
// compares states with EqualVT and logs when the routine exits without being
// canceled.
//
// Note: routines won't start until SetContext and SetState are called.
func NewStateRoutineContainerWithLoggerVT[T proto.EqualVT[T]](le *logrus.Entry, opts ...Option) *StateRoutineContainer[T] {
	return &StateRoutineContainer[T]{
		rc:      NewRoutineContainerWithLogger(le, opts...),
		compare: proto.CompareEqualVT[T](),
	}
}

// GetState returns the immediate state in the StateRoutineContainer.
func (s *StateRoutineContainer[T]) GetState() T {
	var state T
	s.rc.transition(func() (Routine, bool) {
		state = s.s
		return nil, false
	})
	return state
}

// SetState sets the state in the StateRoutineContainer.
//
// Returns if the state changed and if the routine is running.
// If reset=true the existing routine was canceled or restarted.
func (s *StateRoutineContainer[T]) SetState(state T) (waitReturn <-chan struct{}, changed, reset, running bool) {
	waitReturn, reset, running = s.rc.transition(func() (Routine, bool) {
		changed = s.compare == nil || !s.compare(s.s, state)
		if !changed {
			return nil, false
		}
		s.s = state
		return s.routineLocked(), true
	})
	// An unchanged state reports running=false, as callers rely on.
	running = running && changed
	return
}

// SwapValue locks the container, calls the callback, and stores the returned state.
//
// Returns the updated value and if the state changed.
// If reset=true returns a channel which closes when the previous instance has exited.
func (s *StateRoutineContainer[T]) SwapValue(cb func(val T) T) (nextState T, waitReturn <-chan struct{}, changed, reset, running bool) {
	var equal bool
	waitReturn, reset, running = s.rc.transition(func() (Routine, bool) {
		// A nil callback or the same value keeps the state.
		nextState = s.s
		if cb == nil {
			return nil, false
		}
		next := cb(nextState)
		if next == nextState {
			return nil, false
		}

		// A value compare finds equal keeps the state too.
		equal = s.compare != nil && s.compare(nextState, next)
		if equal {
			return nil, false
		}

		// Store the new state and restart the routine with it.
		changed = true
		nextState, s.s = next, next
		return s.routineLocked(), true
	})
	// A new value that compare finds equal reports running=false, as SetState
	// does for an unchanged state.
	running = running && !equal
	return
}

// SetStateRoutine sets the routine to execute, resetting the existing, if set.
// If the specified routine is nil, shuts down the current routine.
// Returns if the current routine was stopped or overwritten.
// Returns a channel which will be closed when every previous routine has exited,
// including one stopped by an earlier nil routine. It is nil if none remain.
// If SetContext has not been called or SetState is empty, returns false for running.
// Note: does not check if routine is equal to the current routine func (cannot compare generic funcs).
func (s *StateRoutineContainer[T]) SetStateRoutine(routine StateRoutine[T]) (waitReturn <-chan struct{}, reset, running bool) {
	return s.rc.transition(func() (Routine, bool) {
		s.stateRoutine = routine
		return s.routineLocked(), true
	})
}

// routineLocked binds the state routine to the current state. It returns nil,
// which stops the routine, when either is unset. Call it inside a transition.
func (s *StateRoutineContainer[T]) routineLocked() Routine {
	var empty T
	st, routine := s.s, s.stateRoutine
	if routine == nil || st == empty {
		return nil
	}
	return func(ctx context.Context) error {
		return routine(ctx, st)
	}
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
// errCh is an optional error channel (can be nil).
func (s *StateRoutineContainer[T]) WaitExited(ctx context.Context, returnIfNotRunning bool, errCh <-chan error) error {
	return s.rc.WaitExited(ctx, returnIfNotRunning, errCh)
}
