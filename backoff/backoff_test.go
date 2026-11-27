package backoff

import (
	"testing"
	"time"

	cbackoff "github.com/aperturerobotics/util/backoff/cbackoff"
)

// TestConstructExponential checks that construction applies the protobuf
// defaults and options before the backoff starts: the first interval is the
// initial interval and elapsed time starts at construction.
func TestConstructExponential(t *testing.T) {
	cases := []struct {
		name    string
		conf    *Backoff
		initial time.Duration
		max     time.Duration
		elapsed time.Duration
	}{
		{"empty", &Backoff{}, 800 * time.Millisecond, 20 * time.Second, 0},
		{"configured", &Backoff{
			BackoffKind: BackoffKind_BackoffKind_EXPONENTIAL,
			Exponential: &Exponential{
				InitialInterval: 100,
				Multiplier:      2,
				MaxInterval:     300,
				MaxElapsedTime:  60000,
			},
		}, 100 * time.Millisecond, 300 * time.Millisecond, time.Minute},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expo, ok := c.conf.Construct().(*cbackoff.ExponentialBackOff)
			if !ok {
				t.Fatal("expected an exponential backoff")
			}
			if expo.MaxElapsedTime != c.elapsed {
				t.Fatalf("max elapsed time %v, want %v", expo.MaxElapsedTime, c.elapsed)
			}
			if expo.RandomizationFactor != 0 {
				t.Fatalf("randomization factor %v, want 0", expo.RandomizationFactor)
			}
			if elapsed := expo.GetElapsedTime(); elapsed < 0 || elapsed > time.Second {
				t.Fatalf("elapsed time %v does not start at construction", elapsed)
			}
			if next := expo.NextBackOff(); next != c.initial {
				t.Fatalf("first interval %v, want %v", next, c.initial)
			}
			for range 8 {
				expo.NextBackOff()
			}
			if next := expo.NextBackOff(); next != c.max {
				t.Fatalf("capped interval %v, want %v", next, c.max)
			}
		})
	}
}
