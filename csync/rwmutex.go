package csync

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/aperturerobotics/util/broadcast"
	"github.com/pkg/errors"
)

// RWMutex implements a RWMutex with a Broadcast.
// Implements a RWMutex that accepts a Context.
// An empty value RWMutex{} is valid.
type RWMutex struct {
	// bcast is broadcast when below fields change
	bcast broadcast.Broadcast
	// nreaders is the number of active readers
	nreaders int
	// writing indicates there's a write tx active
	writing bool
	// writeWaiting indicates the number of waiting write tx
	writeWaiting int
}

// Lock attempts to hold a lock on the RWMutex.
// Returns a lock release function or an error.
// A single writer OR many readers can hold Lock at a time.
// If a writer is waiting to lock, readers will wait for it.
func (m *RWMutex) Lock(ctx context.Context, write bool) (func(), error) {
	// status:
	// 0: waiting for lock
	// 1: locked
	// 2: unlocked (released)
	var status atomic.Int32
	var waitCh <-chan struct{}
	m.bcast.HoldLock(func(_ func(), getWaitCh func() <-chan struct{}) {
		if write {
			if m.nreaders != 0 || m.writing {
				m.writeWaiting++
				waitCh = getWaitCh()
			} else {
				m.writing = true
				status.Store(1)
			}
		} else if !m.writing && m.writeWaiting == 0 {
			m.nreaders++
			status.Store(1)
		} else {
			waitCh = getWaitCh()
		}
	})

	release := func() {
		pre := status.Swap(2)
		if pre == 2 {
			return
		}

		m.bcast.HoldLock(func(broadcast func(), _ func() <-chan struct{}) {
			if pre == 0 {
				// 0: waiting for lock
				if write {
					m.writeWaiting--
				}
			} else {
				// 1: we have the lock
				if write {
					m.writing = false
				} else {
					m.nreaders--
				}
				broadcast()
			}
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
			if write {
				if m.nreaders == 0 && !m.writing {
					m.writeWaiting--
					m.writing = true
					status.Store(1)
				} else {
					waitCh = getWaitCh()
				}
			} else if !m.writing && m.writeWaiting == 0 {
				m.nreaders++
				status.Store(1)
			} else {
				waitCh = getWaitCh()
			}
		})

		if status.Load() == 1 {
			return release, nil
		}
	}
}

// TryLock attempts to hold a lock on the RWMutex.
// Returns a lock release function or nil if the lock could not be grabbed.
// A single writer OR many readers can hold Lock at a time.
// If a writer is waiting to lock, readers will wait for it.
func (m *RWMutex) TryLock(write bool) (func(), bool) {
	var unlocked atomic.Bool
	m.bcast.HoldLock(func(broadcast func(), getWaitCh func() <-chan struct{}) {
		if write {
			if m.nreaders != 0 || m.writing {
				unlocked.Store(true)
			} else {
				m.writing = true
			}
		} else if !m.writing && m.writeWaiting == 0 {
			m.nreaders++
		} else {
			unlocked.Store(true)
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
			if write {
				m.writing = false
			} else {
				m.nreaders--
			}
			broadcast()
		})
	}, true
}

// Locker returns an RWMutexLocker that uses context.Background to write lock the RWMutex.
func (m *RWMutex) Locker() sync.Locker {
	return &RWMutexLocker{m: m, write: true}
}

// RLocker returns an RWMutexLocker that uses context.Background to read lock the RWMutex.
func (m *RWMutex) RLocker() sync.Locker {
	return &RWMutexLocker{m: m, write: false}
}

// RWMutexLocker implements Locker for a RWMutex.
type RWMutexLocker struct {
	m     *RWMutex
	write bool
	mtx   sync.Mutex
	rels  []func()
}

// Lock implements the sync.Locker interface.
func (l *RWMutexLocker) Lock() {
	release, err := l.m.Lock(context.Background(), l.write)
	if err != nil {
		panic(errors.Wrap(err, "csync: failed RWMutexLocker Lock"))
	}
	l.mtx.Lock()
	l.rels = append(l.rels, release)
	l.mtx.Unlock()
}

// Unlock implements the sync.Locker interface.
func (l *RWMutexLocker) Unlock() {
	l.mtx.Lock()
	if len(l.rels) == 0 {
		l.mtx.Unlock()
		panic("csync: unlock of unlocked RWMutexLocker")
	}
	rel := l.rels[len(l.rels)-1]
	if len(l.rels) == 1 {
		l.rels = nil
	} else {
		l.rels[len(l.rels)-1] = nil
		l.rels = l.rels[:len(l.rels)-1]
	}
	l.mtx.Unlock()
	rel()
}

// _ is a type assertion
var _ sync.Locker = ((*RWMutexLocker)(nil))
