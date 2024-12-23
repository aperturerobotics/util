package retry

import (
	"context"
	"time"

	bo "github.com/aperturerobotics/util/backoff"
	"github.com/aperturerobotics/util/backoff/cbackoff"
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
// If the process returns nil or context canceled, it exits.
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

		b := bo.NextBackOff()
		le.
			WithError(err).
			WithField("backoff", b.String()).
			Warn("process failed, retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b):
		}
	}
}
