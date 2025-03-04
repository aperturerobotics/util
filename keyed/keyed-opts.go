package keyed

import (
	"time"

	"github.com/aperturerobotics/util/backoff"
	cbackoff "github.com/aperturerobotics/util/backoff/cbackoff"
	"github.com/sirupsen/logrus"
)

// Option is an option for a Keyed instance.
type Option[K comparable, V any] interface {
	// ApplyToKeyed applies the option to the Keyed.
	ApplyToKeyed(k *Keyed[K, V])
}

type option[K comparable, V any] struct {
	cb func(k *Keyed[K, V])
}

// newOption constructs a new option.
func newOption[K comparable, V any](cb func(k *Keyed[K, V])) *option[K, V] {
	return &option[K, V]{cb: cb}
}

// ApplyToKeyed applies the option to the Keyed instance.
func (o *option[K, V]) ApplyToKeyed(k *Keyed[K, V]) {
	if o.cb != nil {
		o.cb(k)
	}
}

// WithReleaseDelay adds a delay after removing a key before canceling the routine.
func WithReleaseDelay[K comparable, V any](delay time.Duration) Option[K, V] {
	if delay < 0 {
		delay *= -1
	}
	return newOption(func(k *Keyed[K, V]) {
		k.releaseDelay = delay
	})
}

// WithRetry adds a retry after a routine exits with an error.
//
// If the backoff config is nil, disables retry.
func WithRetry[K comparable, V any](bo *backoff.Backoff) Option[K, V] {
	return newOption(func(k *Keyed[K, V]) {
		if bo == nil {
			k.backoffFactory = nil
		} else {
			k.backoffFactory = func(k K) cbackoff.BackOff {
				return bo.Construct()
			}
		}
	})
}

// WithBackoff adds a function to call to construct a backoff.
//
// If the function returns nil, disables retry.
func WithBackoff[K comparable, V any](cb func(k K) cbackoff.BackOff) Option[K, V] {
	return newOption(func(k *Keyed[K, V]) {
		k.backoffFactory = cb
	})
}

// WithExitCb adds a callback after a routine exits.
func WithExitCb[K comparable, V any](cb func(key K, routine Routine, data V, err error)) Option[K, V] {
	return newOption(func(k *Keyed[K, V]) {
		k.exitedCbs = append(k.exitedCbs, cb)
	})
}

// WithExitLogger adds a exited callback which logs information about the exit.
func WithExitLogger[K comparable, V any](le *logrus.Entry) Option[K, V] {
	return WithExitCb(NewLogExitedCallback[K, V](le))
}

// WithExitLoggerWithName adds a exited callback which logs information about the exit with a name string.
func WithExitLoggerWithName[K comparable, V any](le *logrus.Entry, name string) Option[K, V] {
	return WithExitCb(NewLogExitedCallbackWithName[K, V](le, name))
}

// WithExitLoggerWithNameFn adds a exited callback which logs information about the exit with a name function.
func WithExitLoggerWithNameFn[K comparable, V any](le *logrus.Entry, nameFn func(key K) string) Option[K, V] {
	return WithExitCb(NewLogExitedCallbackWithNameFn[K, V](le, nameFn))
}
