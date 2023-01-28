package memo

import (
	"sync/atomic"
)

// MemoizeFunc memoizes the given function.
func MemoizeFunc[T any](fn func() (T, error)) func() (T, error) {
	var started atomic.Bool
	done := make(chan struct{})
	var result T
	var doneErr error
	return func() (T, error) {
		if !started.Swap(true) {
			defer close(done)
			result, doneErr = fn()
			return result, doneErr
		} else {
			<-done
			return result, doneErr
		}
	}
}
