package broadcast

import (
	"context"
	"errors"
	"sync"
)

// Broadcast implements notifying waiters via a channel.
//
// The zero-value of this struct is valid.
type Broadcast struct {
	mtx sync.Mutex
	ch  chan struct{}
}

// HoldLock locks the mutex and calls the callback.
//
// broadcast closes the wait channel, if any.
// getWaitCh returns a channel that will be closed when broadcast is called.
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

// Wait waits for the cb to return true or an error before returning.
// When the broadcast channel is broadcasted, re-calls cb again to re-check the value.
// cb is called while the mutex is locked.
// Returns false, context.Canceled if ctx is canceled.
// Return nil if and only if cb returned true, nil.
func (c *Broadcast) Wait(ctx context.Context, cb func(broadcast func()) (bool, error)) error {
	if cb == nil || ctx == nil {
		return errors.New("cb and ctx must be set")
	}

	var waitCh <-chan struct{}

	for {
		if ctx.Err() != nil {
			return context.Canceled
		}

		var done bool
		var err error
		c.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			done, err = cb(broadcast)
			if !done && err == nil {
				waitCh = getWaitCh()
			}
		})

		if done || err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-waitCh:
		}
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
