// Package memo runs a function at most once and caches its result.
//
// MemoizeFunc returns a function that calls fn on its first call; concurrent
// and later calls wait for that call and return the same value and error.
// Unlike promise.Once, an error is cached too.
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
