package broadcast

import (
	"context"
	"errors"
)

// HoldLock locks the mutex and calls the callback.
//
// broadcast closes the wait channel, if any. getWaitCh returns a channel that
// will be closed when broadcast is called. Prefer Lock in allocation-sensitive
// paths because it exposes the same operations as methods on a guard and avoids
// constructing callback operation closures.
func (c *Broadcast) HoldLock(cb func(broadcast func(), getWaitCh func() <-chan struct{})) {
	locked := c.Lock()
	defer locked.Unlock()
	cb(c.broadcastLockedFunc(), c.waitChLockedFunc())
}

// TryHoldLock attempts to lock the mutex and call the callback.
//
// It returns true if the lock was acquired and the callback was called.
// Prefer TryLock in allocation-sensitive paths.
func (c *Broadcast) TryHoldLock(cb func(broadcast func(), getWaitCh func() <-chan struct{})) bool {
	locked, ok := c.TryLock()
	if !ok {
		return false
	}
	defer locked.Unlock()
	cb(c.broadcastLockedFunc(), c.waitChLockedFunc())
	return true
}

// HoldLockMaybeAsync locks the mutex and calls the callback if possible.
//
// If the mutex cannot be locked right now, it starts a new goroutine to wait
// for it. This is a compatibility helper for callback-shaped callers; direct
// hot paths should use Lock or TryLock.
func (c *Broadcast) HoldLockMaybeAsync(cb func(broadcast func(), getWaitCh func() <-chan struct{})) {
	holdBroadcastLock := func(lock bool) {
		if lock {
			c.mtx.Lock()
		}
		defer c.mtx.Unlock()
		cb(c.broadcastLockedFunc(), c.waitChLockedFunc())
	}

	if c.mtx.TryLock() {
		holdBroadcastLock(false)
		return
	}
	go holdBroadcastLock(true)
}

// Wait waits for the callback to return true or an error before returning.
//
// When the broadcast channel is broadcasted, Wait calls cb again under the
// broadcast lock to re-check the guarded value. It returns context.Canceled if
// ctx is canceled.
func (c *Broadcast) Wait(ctx context.Context, cb func(broadcast func(), getWaitCh func() <-chan struct{}) (bool, error)) error {
	if cb == nil || ctx == nil {
		return errors.New("cb and ctx must be set")
	}

	for {
		if ctx.Err() != nil {
			return context.Canceled
		}

		var waitCh <-chan struct{}
		var done bool
		var err error
		locked := c.Lock()
		broadcast := c.broadcastLockedFunc()
		getWaitCh := c.waitChLockedFunc()
		done, err = cb(broadcast, getWaitCh)
		if !done && err == nil {
			waitCh = getWaitCh()
		}
		locked.Unlock()

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

// broadcastLockedFunc returns a callback-shaped broadcast operation for
// HoldLock compatibility callers. Keep this as a plain closure rather than a
// bound Locked method value: TinyGo browser wasm has trapped in channel close
// paths reached through bound method callbacks.
func (c *Broadcast) broadcastLockedFunc() func() {
	return func() {
		if c.ch == nil {
			return
		}
		ch := c.ch
		c.ch = nil
		ch.close()
	}
}

// waitChLockedFunc returns a callback-shaped wait subscription operation for
// HoldLock compatibility callers.
func (c *Broadcast) waitChLockedFunc() func() <-chan struct{} {
	return func() <-chan struct{} {
		if c.ch == nil {
			c.ch = newBroadcastWaitCh()
		}
		return c.ch.ch
	}
}
