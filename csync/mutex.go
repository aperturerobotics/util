// Package csync holds locks whose acquisition accepts a Context.
//
// Each lock returns a release function from Lock, or the context's error if
// it ends first, and the empty value of each is valid. Mutex grants the lock
// in arrival order. RWMutex admits one writer or many readers, and a waiting
// writer holds back new readers. PrioMutex grants the oldest waiter of the
// highest priority and lets a holder see that someone waits. Semaphore admits
// weight in arrival order up to a limit that can change while in use.
package csync

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Mutex implements a mutex that accepts a Context.
//
// Waiters acquire the lock in arrival order: a release hands the lock to the
// longest waiting caller, so a caller that releases and locks again queues
// behind the others instead of starving them. An empty value Mutex{} is valid.
type Mutex struct {
	// mtx guards the fields below.
	mtx sync.Mutex
	// locked indicates the mutex is held. It stays set while waiters remain.
	locked bool
	// waiters is the queue of callers waiting for the lock, oldest first.
	waiters []chan struct{}
}

// Lock attempts to hold a lock on the Mutex.
// Returns a lock release function or an error.
func (m *Mutex) Lock(ctx context.Context) (func(), error) {
	// Take a free lock, otherwise queue for it.
	m.mtx.Lock()
	if !m.locked {
		m.locked = true
		m.mtx.Unlock()
		return m.newRelease(), nil
	}
	handoff := make(chan struct{})
	m.waiters = append(m.waiters, handoff)
	m.mtx.Unlock()

	// Wait for a release to hand over the lock.
	select {
	case <-handoff:
		return m.newRelease(), nil
	case <-ctx.Done():
	}

	// Leave the queue, passing on a lock handed over during cancellation.
	m.mtx.Lock()
	select {
	case <-handoff:
		m.unlockLocked()
	default:
		m.waiters = slices.DeleteFunc(m.waiters, func(ch chan struct{}) bool {
			return ch == handoff
		})
	}
	m.mtx.Unlock()
	return nil, context.Canceled
}

// TryLock attempts to hold a lock on the Mutex.
// Returns a lock release function or nil if the lock could not be grabbed.
func (m *Mutex) TryLock() (func(), bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.locked {
		return nil, false
	}
	m.locked = true
	return m.newRelease(), true
}

// newRelease returns the release function of a held lock. Calls after the
// first do nothing.
func (m *Mutex) newRelease() func() {
	var released atomic.Bool
	return func() {
		if released.Swap(true) {
			return
		}
		m.mtx.Lock()
		m.unlockLocked()
		m.mtx.Unlock()
	}
}

// unlockLocked hands the lock to the oldest waiter, or unlocks the mutex when
// none waits. The caller holds mtx.
func (m *Mutex) unlockLocked() {
	if len(m.waiters) == 0 {
		m.locked = false
		return
	}
	next := m.waiters[0]
	m.waiters[0] = nil
	m.waiters = m.waiters[1:]
	close(next)
}

// Locker returns a MutexLocker that uses context.Background to lock the Mutex.
func (m *Mutex) Locker() sync.Locker {
	return &MutexLocker{m: m}
}

// MutexLocker implements Locker for a Mutex.
type MutexLocker struct {
	m   *Mutex
	rel atomic.Pointer[func()]
}

// Lock implements the sync.Locker interface.
func (l *MutexLocker) Lock() {
	release, err := l.m.Lock(context.Background())
	if err != nil {
		panic(errors.Wrap(err, "csync: failed MutexLocker Lock"))
	}
	l.rel.Store(&release)
}

// Unlock implements the sync.Locker interface.
func (l *MutexLocker) Unlock() {
	rel := l.rel.Swap(nil)
	if rel == nil {
		panic("csync: unlock of unlocked MutexLocker")
	}
	(*rel)()
}

// _ is a type assertion
var _ sync.Locker = ((*MutexLocker)(nil))
