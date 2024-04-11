package ccontainer

import (
	"context"

	proto "github.com/aperturerobotics/protobuf-go-lite"
	"github.com/aperturerobotics/util/broadcast"
)

// CContainer is a concurrent container.
type CContainer[T comparable] struct {
	bcast broadcast.Broadcast
	val   T
	equal func(a, b T) bool
}

// NewCContainer builds a CContainer with an initial value.
func NewCContainer[T comparable](val T) *CContainer[T] {
	return &CContainer[T]{val: val}
}

// NewCContainerWithEqual builds a CContainer with an initial value and a comparator.
func NewCContainerWithEqual[T comparable](val T, isEqual func(a, b T) bool) *CContainer[T] {
	return &CContainer[T]{val: val, equal: isEqual}
}

// NewCContainerVT constructs a CContainer that uses VTEqual to check for equality.
func NewCContainerVT[T proto.EqualVT[T]](val T) *CContainer[T] {
	return NewCContainerWithEqual[T](val, proto.CompareEqualVT[T]())
}

// GetValue returns the immediate value of the container.
func (c *CContainer[T]) GetValue() T {
	var val T
	c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		val = c.val
	})
	return val
}

// SetValue sets the ccontainer value.
func (c *CContainer[T]) SetValue(val T) {
	c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		if !c.compare(c.val, val) {
			c.val = val
			broadcast()
		}
	})
}

// SwapValue locks the container, calls the callback, and stores the return value.
//
// Returns the updated value.
// If cb is nil returns the current value without changes.
func (c *CContainer[T]) SwapValue(cb func(val T) T) T {
	var val T
	c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		val = c.val
		if cb != nil {
			val = cb(val)
			if !c.compare(c.val, val) {
				c.val = val
				broadcast()
			}
		}
	})
	return val
}

// WaitValueWithValidator waits for any value that matches the validator in the container.
// errCh is an optional channel to read an error from.
func (c *CContainer[T]) WaitValueWithValidator(
	ctx context.Context,
	valid func(v T) (bool, error),
	errCh <-chan error,
) (T, error) {
	var ok bool
	var err error
	var emptyValue T
	for {
		var val T
		var wake <-chan struct{}
		c.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			val = c.val
			wake = getWaitCh()
		})
		if valid != nil {
			ok, err = valid(val)
		} else {
			ok = !c.compare(val, emptyValue)
			err = nil
		}
		if err != nil {
			return emptyValue, err
		}
		if ok {
			return val, nil
		}

		select {
		case <-ctx.Done():
			return emptyValue, ctx.Err()
		case err, ok := <-errCh:
			if !ok {
				// errCh was non-nil but was closed
				// treat this as context canceled
				return emptyValue, context.Canceled
			}
			if err != nil {
				return emptyValue, err
			}
		case <-wake:
			// woken, value changed
		}
	}
}

// WaitValue waits for any non-nil value in the container.
// errCh is an optional channel to read an error from.
func (c *CContainer[T]) WaitValue(ctx context.Context, errCh <-chan error) (T, error) {
	return c.WaitValueWithValidator(ctx, func(v T) (bool, error) {
		var emptyValue T
		return !c.compare(emptyValue, v), nil
	}, errCh)
}

// WaitValueChange waits for a value that is different than the given.
// errCh is an optional channel to read an error from.
func (c *CContainer[T]) WaitValueChange(ctx context.Context, old T, errCh <-chan error) (T, error) {
	return c.WaitValueWithValidator(ctx, func(v T) (bool, error) {
		return !c.compare(old, v), nil
	}, errCh)
}

// WaitValueEmpty waits for an empty value.
// errCh is an optional channel to read an error from.
func (c *CContainer[T]) WaitValueEmpty(ctx context.Context, errCh <-chan error) error {
	_, err := c.WaitValueWithValidator(ctx, func(v T) (bool, error) {
		var emptyValue T
		return c.compare(emptyValue, v), nil
	}, errCh)
	return err
}

// compare checks of two values are equal
func (c *CContainer[T]) compare(a, b T) bool {
	if a == b {
		return true
	}
	if c.equal != nil && c.equal(a, b) {
		return true
	}
	return false
}

// _ is a type assertion
var _ Watchable[struct{}] = ((*CContainer[struct{}])(nil))
