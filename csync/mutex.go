package csync

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/aperturerobotics/util/broadcast"
	"github.com/pkg/errors"
)

// Mutex implements a mutex with a Broadcast.
// Implements a mutex that accepts a Context.
// An empty value Mutex{} is valid.
type Mutex struct {
	// bcast is broadcast when below fields change
	bcast broadcast.Broadcast
	// locked indicates the mutex is locked
	locked bool
}

// Lock attempts to hold a lock on the Mutex.
// Returns a lock release function or an error.
func (m *Mutex) Lock(ctx context.Context) (func(), error) {
	// status:
	// 0: waiting for lock
	// 1: locked
	// 2: unlocked (released)
	var status atomic.Int32
	var waitCh <-chan struct{}
	m.bcast.HoldLock(func(_ func(), getWaitCh func() <-chan struct{}) {
		if m.locked {
			// keep waiting
			waitCh = getWaitCh()
		} else {
			// 0: waiting for lock
			// 1: have the lock
			swapped := status.CompareAndSwap(0, 1)
			if swapped {
				m.locked = true
			}
		}
	})

	release := func() {
		pre := status.Swap(2)
		// 1: we have the lock
		if pre != 1 {
			return
		}

		// unlock
		m.bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
			m.locked = false
			broadcast()
		})
	}

	// fast path: we locked the mutex
	if status.Load() == 1 {
		return release, nil
	}

	// slow path: watch for changes
	for {
		select {
		case <-ctx.Done():
			release()
			return nil, context.Canceled
		case <-waitCh:
		}

		m.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
			// keep waiting for the lock
			if m.locked {
				waitCh = getWaitCh()
				return
			}

			// 0: waiting for lock
			// 1: have the lock
			swapped := status.CompareAndSwap(0, 1)
			if swapped {
				m.locked = true
			}
		})

		nstatus := status.Load()
		if nstatus == 1 {
			return release, nil
		} else if nstatus == 2 {
			return nil, context.Canceled
		}
	}
}

// TryLock attempts to hold a lock on the Mutex.
// Returns a lock release function or nil if the lock could not be grabbed.
func (m *Mutex) TryLock() (func(), bool) {
	var unlocked atomic.Bool
	m.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		if m.locked {
			unlocked.Store(true)
		} else {
			m.locked = true
		}
	})

	// we failed to lock the mutex
	if unlocked.Load() {
		return nil, false
	}

	return func() {
		if unlocked.Swap(true) {
			return
		}

		m.bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
			m.locked = false
			broadcast()
		})
	}, true
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
