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
func (b *Backoff) constructExpo() backoff.BackOff {
	expo := backoff.NewExponentialBackOff()
	opts := b.GetExponential()

	initialInterval := opts.GetInitialInterval()
	if initialInterval == 0 {
		// default to 800ms
		initialInterval = 800
	}
	expo.InitialInterval = time.Duration(initialInterval) * time.Millisecond

	multiplier := opts.GetMultiplier()
	if multiplier == 0 {
		multiplier = 1.8
	}
	expo.Multiplier = float64(multiplier)

	maxInterval := opts.GetMaxInterval()
	if maxInterval == 0 {
		maxInterval = 20000
	}
	expo.MaxInterval = time.Duration(maxInterval) * time.Millisecond
	expo.RandomizationFactor = float64(opts.GetRandomizationFactor())
	if opts.GetMaxElapsedTime() == 0 {
		expo.MaxElapsedTime = 0
	} else {
		expo.MaxElapsedTime = time.Duration(opts.GetMaxElapsedTime()) * time.Millisecond
	}
	expo.Reset()
	return expo
}

// constructConstant constructs a constant backoff.
func (b *Backoff) constructConstant() backoff.BackOff {
	dur := b.GetConstant().GetInterval()
	if dur == 0 {
		dur = 5000
	}
	bo := backoff.NewConstantBackOff(time.Duration(dur) * time.Millisecond)
	bo.Reset()
	return bo
}
