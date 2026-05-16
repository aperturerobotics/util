package broadcast

// Locked is a held Broadcast lock for allocation-sensitive callers.
//
// Broadcast also has callback helpers, but those helpers pass broadcast and
// wait-channel operations as function values. Hot callers can use Locked to do
// the same work with direct method calls, which avoids per-lock callback
// closures and keeps the state check next to the optional wait subscription.
// Call Unlock exactly once, and do not copy a Locked value after use.
type Locked struct {
	b *Broadcast
}

// Lock locks the broadcast and returns a held lock guard.
//
// Use this API in hot paths that already have clear lock scope. It exists so
// callers can inspect guarded state, call WaitCh only when they must block, and
// call Broadcast without allocating callback operation closures.
func (c *Broadcast) Lock() Locked {
	c.mtx.Lock()
	return Locked{b: c}
}

// TryLock attempts to lock the broadcast and returns whether it succeeded.
func (c *Broadcast) TryLock() (Locked, bool) {
	if !c.mtx.TryLock() {
		return Locked{}, false
	}
	return Locked{b: c}, true
}

// Unlock releases the held broadcast lock.
func (l *Locked) Unlock() {
	l.b.mtx.Unlock()
	l.b = nil
}

// Broadcast closes the current wait channel and starts a new wait epoch.
//
// Call Broadcast while holding the Locked guard after mutating the guarded
// state. Waiters that already called WaitCh wake from the closed channel, and a
// later WaitCh call allocates the next epoch.
func (l *Locked) Broadcast() {
	if l.b.ch == nil {
		return
	}
	ch := l.b.ch
	l.b.ch = nil
	ch.close()
}

// WaitCh returns a channel that closes on the next Broadcast call.
//
// WaitCh allocates the wait epoch lazily, so callers should call it only after
// they have checked the guarded state and determined they really need to block.
func (l *Locked) WaitCh() <-chan struct{} {
	if l.b.ch == nil {
		l.b.ch = newBroadcastWaitCh()
	}
	return l.b.ch.ch
}
