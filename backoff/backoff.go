// Package backoff configures retry backoff with a protobuf message.
//
// A Backoff message selects an exponential or constant kind and its
// intervals in milliseconds; Construct builds the matching cbackoff.BackOff.
// Unset fields take defaults: exponential starts at 800ms, multiplies by 1.8
// and caps at 20s with no elapsed-time limit, and constant waits 5s. An empty
// message constructs the exponential default.
package backoff

import (
	"time"

	backoff "github.com/aperturerobotics/util/backoff/cbackoff"
	"github.com/pkg/errors"
)

// Stop indicates that no more retries should be made for use in NextBackOff().
const Stop = backoff.Stop

// GetEmpty returns if the backoff config is empty.
func (b *Backoff) GetEmpty() bool {
	return b.GetBackoffKind() == 0
}

// Construct constructs the backoff.
// Validates the options.
func (b *Backoff) Construct() backoff.BackOff {
	switch b.GetBackoffKind() {
	default:
		fallthrough
	case BackoffKind_BackoffKind_EXPONENTIAL:
		return b.constructExpo()
	case BackoffKind_BackoffKind_CONSTANT:
		return b.constructConstant()
	}
}

// Validate validates the backoff kind.
func (b BackoffKind) Validate() error {
	switch b {
	case BackoffKind_BackoffKind_UNKNOWN:
	case BackoffKind_BackoffKind_EXPONENTIAL:
	case BackoffKind_BackoffKind_CONSTANT:
	default:
		return errors.Errorf("unknown backoff kind: %s", b.String())
	}
	return nil
}

// Validate validates the backoff config.
func (b *Backoff) Validate(allowEmpty bool) error {
	if !allowEmpty && b.GetEmpty() {
		return errors.New("backoff must be set")
	}
	if err := b.GetBackoffKind().Validate(); err != nil {
		return err
	}
	return nil
}

// constructExpo constructs an exponential backoff.
//
// A zero MaxElapsedTime passes through as zero, which cbackoff treats as no
// elapsed-time limit. The constructor applies the options and then resets, so
// the first interval is the initial interval and elapsed time starts here.
func (b *Backoff) constructExpo() backoff.BackOff {
	// Unset intervals and multiplier take the package defaults.
	opts := b.GetExponential()
	initialInterval := opts.GetInitialInterval()
	if initialInterval == 0 {
		initialInterval = 800
	}
	multiplier := opts.GetMultiplier()
	if multiplier == 0 {
		multiplier = 1.8
	}
	maxInterval := opts.GetMaxInterval()
	if maxInterval == 0 {
		maxInterval = 20000
	}

	// Construction applies the options and starts the backoff.
	return backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Duration(initialInterval)*time.Millisecond),
		backoff.WithMultiplier(float64(multiplier)),
		backoff.WithMaxInterval(time.Duration(maxInterval)*time.Millisecond),
		backoff.WithRandomizationFactor(float64(opts.GetRandomizationFactor())),
		backoff.WithMaxElapsedTime(time.Duration(opts.GetMaxElapsedTime())*time.Millisecond),
	)
}

// constructConstant constructs a constant backoff.
func (b *Backoff) constructConstant() backoff.BackOff {
	// An unset interval waits 5s.
	dur := b.GetConstant().GetInterval()
	if dur == 0 {
		dur = 5000
	}
	bo := backoff.NewConstantBackOff(time.Duration(dur) * time.Millisecond)
	bo.Reset()
	return bo
}
