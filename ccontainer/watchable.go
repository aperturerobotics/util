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
