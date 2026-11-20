// Package retry reruns a function with backoff until it succeeds.
//
// Retry calls the function until it returns nil, ctx ends, or the backoff
// returns Stop, waiting the backoff's interval after each error. The function
// receives a success callback that resets the backoff once it has made
// progress.
package retry

import (
	"context"
	"time"

	bo "github.com/aperturerobotics/util/backoff"
	backoff "github.com/aperturerobotics/util/backoff/cbackoff"
	"github.com/sirupsen/logrus"
)

// NewBackOff constructs a new backoff with a config.
func NewBackOff(conf *bo.Backoff) backoff.BackOff {
	if conf == nil {
		conf = &bo.Backoff{}
	}
	return conf.Construct()
}

// DefaultBackoff returns the default backoff.
func DefaultBackoff() backoff.BackOff {
	return NewBackOff(nil)
}

// Retry uses a backoff to re-try a process.
// If the process returns nil or ctx is canceled, it exits.
// If the backoff returns Stop, it returns the process's last error.
// If bo is nil, a default one is created.
// Success function will reset the backoff.
func Retry(
	ctx context.Context,
	le *logrus.Entry,
	f func(ctx context.Context, success func()) error,
	bo backoff.BackOff,
) error {
	if bo == nil {
		bo = DefaultBackoff()
	}

	for {
		le.Debug("starting process")
		err := f(ctx, bo.Reset)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err == nil {
			return nil
		}

		// Give up once the backoff says to stop.
		b := bo.NextBackOff()
		if b == backoff.Stop {
			return err
		}
		le.
			WithError(err).
			WithField("backoff", b.String()).
			Warn("process failed, retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b): //gostyle:timer backoff
		}
	}
}
