package ccontainer

import "context"

// Watchable is an interface implemented by ccontainer for watching a value.
type Watchable[T comparable] interface {
	// GetValue returns the current value.
	GetValue() T
	// WaitValueWithValidator waits for any value that matches the validator in the container.
	// errCh is an optional channel to read an error from.
	WaitValueWithValidator(
		ctx context.Context,
		valid func(v T) (bool, error),
		errCh <-chan error,
	) (T, error)

	// WaitValue waits for any non-nil value in the container.
	// errCh is an optional channel to read an error from.
	WaitValue(ctx context.Context, errCh <-chan error) (T, error)
	// WaitValueChange waits for a value that is different than the given.
	// errCh is an optional channel to read an error from.
	WaitValueChange(ctx context.Context, old T, errCh <-chan error) (T, error)
	// WaitValueEmpty waits for an empty value.
	// errCh is an optional channel to read an error from.
	WaitValueEmpty(ctx context.Context, errCh <-chan error) error
}

// ToWatchable converts a ccontainer to a Watchable (somewhat read-only).
func ToWatchable[T comparable](ctr *CContainer[T]) Watchable[T] {
	return ctr
}

// WatchChanges watches a Watchable and calls the callback when the value
// changes. Note: the value pointer must change to trigger an update.
//
// initial is the initial value to wait for changes on.
// set initial to nil to wait for value != nil.
//
// T is the type of the message.
// errCh is an optional error channel to interrupt the operation.
func WatchChanges[T comparable](
	ctx context.Context,
	initialVal T,
	ctr Watchable[T],
	updateCb func(msg T) error,
	errCh <-chan error,
) error {
	// watch for changes
	current := initialVal
	for {
		next, err := ctr.WaitValueChange(ctx, current, errCh)
		if err != nil {
			return err
		}

		current = next
		if err := updateCb(next); err != nil {
			return err
		}
	}
}
