package broadcast

import (
	"context"
	"testing"
)

func TestWatchBroadcast_SendsInitialValue(t *testing.T) {
	var bcast Broadcast
	val := 42

	ctx, cancel := context.WithCancel(context.Background())
	var received []int
	err := WatchBroadcast(ctx, &bcast, func() int { return val }, func(v int) error {
		received = append(received, v)
		cancel()
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(received) != 1 || received[0] != 42 {
		t.Fatalf("expected [42], got %v", received)
	}
}

func TestWatchBroadcast_SkipsDuplicates(t *testing.T) {
	var bcast Broadcast
	val := 1

	ctx, cancel := context.WithCancel(context.Background())
	var received []int
	sent := make(chan struct{}, 10)

	done := make(chan error, 1)
	go func() {
		done <- WatchBroadcast(ctx, &bcast, func() int { return val }, func(v int) error {
			received = append(received, v)
			sent <- struct{}{}
			return nil
		})
	}()

	// Wait for initial send.
	<-sent

	// Broadcast with same value (should be skipped).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		broadcast()
	})

	// Change value and broadcast (should be sent).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = 2
		broadcast()
	})

	// Wait for second send.
	<-sent
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(received) != 2 || received[0] != 1 || received[1] != 2 {
		t.Fatalf("expected [1, 2], got %v", received)
	}
}

func TestWatchBroadcastWithEqual_CustomComparator(t *testing.T) {
	type pair struct{ a, b int }

	var bcast Broadcast
	val := pair{1, 2}

	ctx, cancel := context.WithCancel(context.Background())
	var received []pair
	sent := make(chan struct{}, 10)

	done := make(chan error, 1)
	go func() {
		done <- WatchBroadcastWithEqual(
			ctx, &bcast,
			func() pair { return val },
			func(v pair) error {
				received = append(received, v)
				sent <- struct{}{}
				return nil
			},
			// Only compare .a field.
			func(x, y pair) bool { return x.a == y.a },
		)
	}()

	// Wait for initial send.
	<-sent

	// Change .b only (should be skipped by custom equal).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = pair{1, 99}
		broadcast()
	})

	// Change .a (should be sent).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = pair{2, 99}
		broadcast()
	})

	<-sent
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("expected 2 sends, got %d: %v", len(received), received)
	}
	if received[0].a != 1 || received[1].a != 2 {
		t.Fatalf("expected a=1 then a=2, got %v", received)
	}
}

// vtMsg implements proto.EqualVT for testing WatchBroadcastVT.
type vtMsg struct {
	data string
}

func (m *vtMsg) EqualVT(other *vtMsg) bool {
	if m == nil && other == nil {
		return true
	}
	if m == nil || other == nil {
		return false
	}
	return m.data == other.data
}

func TestWatchBroadcastVT_SkipsDuplicates(t *testing.T) {
	var bcast Broadcast
	val := &vtMsg{data: "hello"}

	ctx, cancel := context.WithCancel(context.Background())
	var received []*vtMsg
	sent := make(chan struct{}, 10)

	done := make(chan error, 1)
	go func() {
		done <- WatchBroadcastVT(
			ctx, &bcast,
			func() *vtMsg { return val },
			func(v *vtMsg) error {
				received = append(received, v)
				sent <- struct{}{}
				return nil
			},
		)
	}()

	// Wait for initial send.
	<-sent

	// Different pointer, same data (should be skipped by VT equal).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = &vtMsg{data: "hello"}
		broadcast()
	})

	// Different data (should be sent).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = &vtMsg{data: "world"}
		broadcast()
	})

	<-sent
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("expected 2 sends, got %d: %v", len(received), received)
	}
	if received[0].data != "hello" || received[1].data != "world" {
		t.Fatalf("expected hello then world, got %v %v", received[0].data, received[1].data)
	}
}

func TestWatchBroadcastVT_NilToNonNil(t *testing.T) {
	var bcast Broadcast
	var val *vtMsg

	ctx, cancel := context.WithCancel(context.Background())
	var received []*vtMsg
	sent := make(chan struct{}, 10)

	done := make(chan error, 1)
	go func() {
		done <- WatchBroadcastVT(
			ctx, &bcast,
			func() *vtMsg { return val },
			func(v *vtMsg) error {
				received = append(received, v)
				sent <- struct{}{}
				return nil
			},
		)
	}()

	// Wait for initial send (nil).
	<-sent

	// Broadcast nil -> non-nil (should send).
	bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
		val = &vtMsg{data: "first"}
		broadcast()
	})

	<-sent
	cancel()

	err := <-done
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("expected 2 sends (nil then non-nil), got %d", len(received))
	}
	if received[0] != nil {
		t.Fatalf("expected first send to be nil, got %v", received[0])
	}
	if received[1].data != "first" {
		t.Fatalf("expected second send to be 'first', got %v", received[1].data)
	}
}
