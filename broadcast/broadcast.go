package broadcast

import "sync"

// adapted from missinggo (MIT license)
// https://github.com/anacrolix/missinggo/blob/master/chancond.go

// Broadcast implements notifying waiters via a channel.
//
// The zero-value of this struct is valid.
type Broadcast struct {
	mtx sync.Mutex
	ch  chan struct{}
}

// GetWaitCh returns a channel that is closed when Broadcast is called.
func (c *Broadcast) GetWaitCh() <-chan struct{} {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.getWaitChLocked()
}

// Broadcast closes the broadcast channel, triggering waiters.
func (c *Broadcast) Broadcast() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.broadcastLocked()
}

// HoldLock locks the mutex and calls the callback.
func (c *Broadcast) HoldLock(cb func(broadcast func(), getWaitCh func() <-chan struct{})) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	cb(c.broadcastLocked, c.getWaitChLocked)
}

// HoldLockMaybeAsync locks the mutex and calls the callback if possible.
// If the mutex cannot be locked right now, starts a new Goroutine to wait for it.
func (c *Broadcast) HoldLockMaybeAsync(cb func(broadcast func(), getWaitCh func() <-chan struct{})) {
	holdBroadcastLock := func(lock bool) {
		if lock {
			c.mtx.Lock()
		}
		// use defer to catch panic cases
		defer c.mtx.Unlock()
		cb(c.broadcastLocked, c.getWaitChLocked)
	}

	// fast path: lock immediately
	if c.mtx.TryLock() {
		holdBroadcastLock(false)
	} else {
		// slow path: use separate goroutine
		go holdBroadcastLock(true)
	}
}

// broadcastLocked is the implementation of Broadcast while mtx is locked.
func (c *Broadcast) broadcastLocked() {
	if c.ch != nil {
		close(c.ch)
		c.ch = nil
	}
}

// getWaitChLocked is the implementation of GetWaitCh while mtx is locked.
func (c *Broadcast) getWaitChLocked() <-chan struct{} {
	if c.ch == nil {
		c.ch = make(chan struct{})
	}
	return c.ch
}
