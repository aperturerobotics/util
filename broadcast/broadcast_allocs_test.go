//go:build !goscript

package broadcast

import "testing"

func TestBroadcastLockDoesNotAllocate(t *testing.T) {
	var bc Broadcast

	allocs := testing.AllocsPerRun(100, func() {
		locked := bc.Lock()
		locked.Broadcast()
		locked.Unlock()
	})
	if allocs != 0 {
		t.Fatalf("expected Lock/Broadcast/Unlock to avoid allocations, got %v", allocs)
	}
}
