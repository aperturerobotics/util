package backoff

import (
	"math/rand"
	"testing"
	"time"
)

func TestMaxTries(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	max := 17 + r.Intn(13)
	bo := WithMaxRetries(&ZeroBackOff{}, uint64(max)) //nolint:gosec

	// Load up the tries count, but reset should clear the record
	for range max / 2 {
		bo.NextBackOff()
	}
	bo.Reset()

	// Now fill the tries count all the way up
	for ix := range max {
		d := bo.NextBackOff()
		if d == Stop {
			t.Errorf("returned Stop on try %d", ix)
		}
	}

	// We have now called the BackOff max number of times, we expect
	// the next result to be Stop, even if we try it multiple times
	for range 7 {
		d := bo.NextBackOff()
		if d != Stop {
			t.Error("invalid next back off")
		}
	}

	// Reset makes it all work again
	bo.Reset()
	d := bo.NextBackOff()
	if d == Stop {
		t.Error("returned Stop after reset")
	}
}
