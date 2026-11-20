// Package broadcast wakes waiters when state guarded by a mutex changes.
//
// A Broadcast is a mutex paired with a wait channel. A caller locks it, reads
// or changes the guarded state, and either calls Broadcast to close the
// current channel and start a new epoch, or takes WaitCh to block until the
// next Broadcast. WaitCh allocates lazily, so a caller that need not block
// costs no channel. Lock and TryLock return a Locked guard with method calls
// for hot paths; HoldLock, TryHoldLock and Wait take callbacks. WatchBroadcast
// and its variants push deduplicated snapshots to a sender, and WaitAny waits
// on several channels collected under their owners' locks. The zero value is
// valid.
package broadcast

import (
	"sync"
)

// Broadcast implements notifying waiters via a channel.
//
// The zero-value of this struct is valid.
type Broadcast struct {
	mtx sync.Mutex
	ch  *broadcastWaitCh
}

type broadcastWaitCh struct {
	once sync.Once
	ch   chan struct{}
}

func newBroadcastWaitCh() *broadcastWaitCh {
	return &broadcastWaitCh{ch: make(chan struct{})}
}

func (c *broadcastWaitCh) close() {
	c.once.Do(func() {
		close(c.ch)
	})
}
