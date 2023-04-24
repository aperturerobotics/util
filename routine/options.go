package routine

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
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

// WithBackoff returns an exited callback which restarts the routine after a
// backoff if the routine returned an error.
//
// Resets the backoff if the routine returned successfully.
// le is an optional logger to log the backoff.
func WithBackoff(bo backoff.BackOff, le *logrus.Entry) Option {
	return newOption(func(k *RoutineContainer) {
		k.exitedCbs = append(k.exitedCbs, func(err error) {
			kctx := k.ctx
			if kctx == nil {
				// no context: do nothing
				return
			}
			select {
			case <-k.ctx.Done():
				// context canceled: do nothing
				return
			default:
			}

			if err == nil {
				bo.Reset()
				if le != nil {
					le.Debug("routine exited successfully")
				}
				return
			}

			nextBackoff := bo.NextBackOff()
			if nextBackoff == backoff.Stop {
				if le != nil {
					le.WithError(err).Warn("routine failed and backoff attempts exceeded")
				}
				return
			}
			if le != nil {
				le.
					WithError(err).
					WithField("backoff-dur", nextBackoff.String()).
					Warn("routine failed: backing off before restart")
			}
			wait := k.bcast.GetWaitCh()
			go func() {
				tmr := time.NewTimer(nextBackoff)
			WaitLoop:
				for {
					select {
					case <-kctx.Done():
						_ = tmr.Stop()
						return
					case <-tmr.C:
						break WaitLoop
					case <-wait:
						k.mtx.Lock()
						if k.ctx != kctx || k.routine == nil || !k.routine.exited {
							wait = nil
						} else {
							wait = k.bcast.GetWaitCh()
						}
						k.mtx.Unlock()
						if wait == nil {
							_ = tmr.Stop()
							return
						}
					}
				}
				k.mtx.Lock()
				if k.ctx == kctx {
					_ = k.restartRoutineLocked(true)
				}
				k.mtx.Unlock()
			}()
		})
	})
}
