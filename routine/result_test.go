package routine

import (
	"context"
	"testing"

	"github.com/aperturerobotics/util/promise"
)

// TestStateResultRoutine tests the state result routine functionality
func TestStateResultRoutine(t *testing.T) {
	ctx := context.Background()

	// Test successful case
	sr, ctr := NewStateResultRoutine(func(ctx context.Context, st int) (string, error) {
		return "value:" + string(rune(st+'0')), nil
	})

	// Set initial state and check result
	if err := sr(ctx, 1); err != nil {
		t.Fatal(err)
	}

	prom, _ := ctr.GetPromise()
	res, err := prom.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res != "value:1" {
		t.Fatalf("expected value:1 got %v", res)
	}

	// Test with custom promise container
	customCtr := promise.NewPromiseContainer[string]()
	sr3 := NewStateResultRoutineWithPromiseContainer(func(ctx context.Context, st int) (string, error) {
		return "custom:" + string(rune(st+'0')), nil
	}, customCtr)

	if err := sr3(ctx, 2); err != nil {
		t.Fatal(err)
	}

	prom3, _ := customCtr.GetPromise()
	res, err = prom3.Await(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res != "custom:2" {
		t.Fatalf("expected custom:2 got %v", res)
	}
}
