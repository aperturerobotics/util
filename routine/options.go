package routine

import (
	"context"

	"github.com/aperturerobotics/util/backoff"
	cbackoff "github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

// Option is an option for a RoutineContainer instance.
type Option interface {
	// ApplyToRoutineContainer applies the option to the RoutineContainer.
	ApplyToRoutineContainer(k *RoutineContainer)
}

type option struct {
	cb func(k *RoutineContainer)
}

// newOption constructs a new option.
func newOption(cb func(k *RoutineContainer)) *option {
	return &option{cb: cb}
}

// ApplyToRoutineContainer applies the option to the RoutineContainer instance.
func (o *option) ApplyToRoutineContainer(k *RoutineContainer) {
	if o.cb != nil {
		o.cb(k)
	}
}

// WithExitCb adds a callback after a routine exits.
func WithExitCb(cb func(err error)) Option {
	return newOption(func(k *RoutineContainer) {
		k.exitedCbs = append(k.exitedCbs, cb)
	})
}

// WithExitLogger adds a exited callback which logs information about the exit.
func WithExitLogger(le *logrus.Entry) Option {
	return WithExitCb(NewLogExitedCallback(le))
}

// NewLogExitedCallback returns a ExitedCb which logs when a controller exited.
func NewLogExitedCallback(le *logrus.Entry) func(err error) {
	return func(err error) {
		if err != nil && err != context.Canceled {
			le.WithError(err).Warnf("routine exited")
		} else {
			le.Debug("routine exited")
		}
	}
}

// WithRetry configures a backoff configuration to use when the routine returns an error.
//
// resets the backoff if the routine returned successfully.
// disables the backoff if config is nil
func WithRetry(boConf *backoff.Backoff) Option {
	return newOption(func(k *RoutineContainer) {
		if boConf == nil {
			k.retryBo = nil
			return
		}

		k.retryBo = boConf.Construct()
	})
}

// WithBackoff configures a backoff to use when the routine returns an error.
//
// resets the backoff if the routine returned successfully.
// le is an optional logger to log the backoff.
// disables the backoff if bo = nil
func WithBackoff(bo cbackoff.BackOff) Option {
	return newOption(func(k *RoutineContainer) {
		k.retryBo = bo
	})
}
