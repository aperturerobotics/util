package routine

import (
	"context"

	"github.com/aperturerobotics/util/promise"
)

// StateResultRoutine is a function called as a goroutine with a state parameter.
// If the state changes, ctx will be canceled and the function restarted.
// If nil is returned as first return value, exits cleanly permanently.
// If an error is returned, can still be restarted later.
// The second return value is the result value.
// This is a wrapper around StateRoutine that also returns a result.
type StateResultRoutine[T comparable, R any] func(ctx context.Context, st T) (R, error)

// NewStateResultRoutine constructs a new StateRoutine from a StateResultRoutine.
// The routine stores the result in the PromiseContainer.
func NewStateResultRoutine[T comparable, R any](srr StateResultRoutine[T, R]) (StateRoutine[T], *promise.PromiseContainer[R]) {
	ctr := promise.NewPromiseContainer[R]()
	return NewStateResultRoutineWithPromiseContainer(srr, ctr), ctr
}

// NewStateResultRoutineWithPromiseContainer constructs a new StateRoutine from a StateResultRoutine.
// The routine stores the result in the provided PromiseContainer.
func NewStateResultRoutineWithPromiseContainer[T comparable, R any](
	srr StateResultRoutine[T, R],
	resultCtr *promise.PromiseContainer[R],
) StateRoutine[T] {
	return func(ctx context.Context, st T) error {
		prom := promise.NewPromise[R]()
		resultCtr.SetPromise(prom)

		result, err := srr(ctx, st)
		if ctx.Err() != nil {
			return context.Canceled
		}
		prom.SetResult(result, err)

		return err
	}
}
