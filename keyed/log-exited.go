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
