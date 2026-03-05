package broadcast

import (
	"context"

	proto "github.com/aperturerobotics/protobuf-go-lite"
)

// WatchBroadcast watches a broadcast for changes and sends snapshots.
//
// snapshot is called under the broadcast lock to get the current value.
// send is called outside the lock to transmit the value.
// Skips sending when the value is equal to the previous via ==.
// Returns when ctx is canceled or send returns an error.
func WatchBroadcast[T comparable](
	ctx context.Context,
	bcast *Broadcast,
	snapshot func() T,
	send func(T) error,
) error {
	return WatchBroadcastWithEqual(ctx, bcast, snapshot, send, nil)
}

// WatchBroadcastWithEqual watches a broadcast for changes and sends snapshots.
//
// snapshot is called under the broadcast lock to get the current value.
// send is called outside the lock to transmit the value.
// equal is an optional comparator; if nil, uses == for dedup.
// Returns when ctx is canceled or send returns an error.
func WatchBroadcastWithEqual[T comparable](
	ctx context.Context,
	bcast *Broadcast,
	snapshot func() T,
	send func(T) error,
	equal func(a, b T) bool,
) error {
	var ch <-chan struct{}
	var val T
	bcast.HoldLock(func(_ func(), getWaitCh func() <-chan struct{}) {
		ch = getWaitCh()
		val = snapshot()
	})
	if err := send(val); err != nil {
		return err
	}
	prev := val
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
		}
		bcast.HoldLock(func(_ func(), getWaitCh func() <-chan struct{}) {
			ch = getWaitCh()
			val = snapshot()
		})
		if val == prev {
			continue
		}
		if equal != nil && equal(val, prev) {
			continue
		}
		if err := send(val); err != nil {
			return err
		}
		prev = val
	}
}

// WatchBroadcastVT watches a broadcast for changes and sends snapshots.
//
// Uses EqualVT for deduplication. Same as WatchBroadcast but for VTProtobuf messages.
func WatchBroadcastVT[T proto.EqualVT[T]](
	ctx context.Context,
	bcast *Broadcast,
	snapshot func() T,
	send func(T) error,
) error {
	return WatchBroadcastWithEqual(ctx, bcast, snapshot, send, proto.CompareEqualVT[T]())
}
