package keyed

import (
	"context"

	"github.com/sirupsen/logrus"
)

// NewLogExitedCallback returns a ExitedCb which logs when a controller exited.
func NewLogExitedCallback[K comparable, V any](le *logrus.Entry) func(key K, routine Routine, data V, err error) {
	return func(key K, routine Routine, data V, err error) {
		if err != nil && err != context.Canceled {
			le.WithError(err).Warnf("keyed: routine exited: %v", key)
		} else {
			le.Debugf("keyed: routine exited: %v", key)
		}
	}
}

// NewLogExitedCallbackWithName returns a ExitedCb which logs when a controller exited with a name instead of the key.
func NewLogExitedCallbackWithName[K comparable, V any](le *logrus.Entry, name string) func(key K, routine Routine, data V, err error) {
	return func(key K, routine Routine, data V, err error) {
		if err != nil && err != context.Canceled {
			le.WithError(err).Warnf("keyed: routine exited: %v", name)
		} else {
			le.Debugf("keyed: routine exited: %v", name)
		}
	}
}

// NewLogExitedCallbackWithNameFn returns a ExitedCb which logs when a controller exited with a name function.
func NewLogExitedCallbackWithNameFn[K comparable, V any](le *logrus.Entry, nameFn func(key K) string) func(key K, routine Routine, data V, err error) {
	if nameFn == nil {
		return NewLogExitedCallback[K, V](le)
	}

	return func(key K, routine Routine, data V, err error) {
		if err != nil && err != context.Canceled {
			le.WithError(err).Warnf("keyed: routine exited: %v", nameFn(key))
		} else {
			le.Debugf("keyed: routine exited: %v", nameFn(key))
		}
	}
}
