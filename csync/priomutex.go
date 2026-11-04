package csync

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"sync/atomic"
)

// PrioMutex implements a mutex that accepts a Context and grants the lock by
// priority.
//
// A release hands the lock to the oldest waiter of the highest waiting
// priority, so waiters of one priority acquire it in arrival order and a
// higher priority waiter goes before every lower one, whenever it arrived. A
// release wakes only the waiter it grants, and costs the same however many
// callers wait.
//
// PrioMutex cannot preempt a holder: a high priority waiter still waits for
// the current holder to release. A holder that runs long bounds that wait by
// checking Waiting at safe points and releasing when it reports true. Lower
// priority waiters can starve while higher priority callers keep the lock
// busy. An empty value PrioMutex{} is valid.
type PrioMutex struct {
	// mtx guards the fields below except top, and is never held across a wait.
	mtx sync.Mutex
	// locked indicates the mutex is held. It stays set while waiters remain.
	locked bool
	// classes holds one queue per waiting priority in ascending priority
	// order. A class is removed as soon as its queue empties, so the last
	// class is the highest priority with a waiter.
	classes []*prioClass
	// top is the last entry of classes, or nil when no caller waits. It is
	// read without mtx by Waiting.
	top atomic.Pointer[prioClass]
}

// prioClass is the queue of callers waiting at one priority.
type prioClass struct {
	// prio is the priority of the callers in waiters.
	prio int
	// waiters is the queue of callers, oldest first. Closing a channel grants
	// that caller the lock.
	waiters []chan struct{}
}

// Lock attempts to hold a lock on the PrioMutex, ahead of every waiter with a
// lower prio. Returns a lock release function, or ctx's error if ctx ends
// before the lock is granted.
func (m *PrioMutex) Lock(ctx context.Context, prio int) (func(), error) {
	// Take a free lock, otherwise queue in the class of prio.
	m.mtx.Lock()
	if !m.locked {
		m.locked = true
		m.mtx.Unlock()
		return m.newRelease(), nil
	}
	class := m.classLocked(prio)
	handoff := make(chan struct{})
	class.waiters = append(class.waiters, handoff)
	m.mtx.Unlock()

	// Wait for a release to hand over the lock.
	select {
	case <-handoff:
		return m.newRelease(), nil
	case <-ctx.Done():
	}

	// Leave the queue, passing on a lock handed over during cancellation.
	m.mtx.Lock()
	if i := slices.Index(class.waiters, handoff); i >= 0 {
		class.waiters = slices.Delete(class.waiters, i, i+1)
		if len(class.waiters) == 0 {
			m.removeClassLocked(class)
		}
	} else {
		m.unlockLocked()
	}
	m.mtx.Unlock()
	return nil, ctx.Err()
}

// TryLock attempts to hold a lock on the PrioMutex.
// Returns a lock release function or nil if the lock could not be grabbed.
func (m *PrioMutex) TryLock() (func(), bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.locked {
		return nil, false
	}
	m.locked = true
	return m.newRelease(), true
}

// Waiting reports whether a caller of priority prio or higher waits for the
// lock. It reads one atomic value, so a holder can call it often.
func (m *PrioMutex) Waiting(prio int) bool {
	top := m.top.Load()
	return top != nil && top.prio >= prio
}

// newRelease returns the release function of a held lock. Calls after the
// first do nothing.
func (m *PrioMutex) newRelease() func() {
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

// classLocked returns the class of prio, adding it in order when no caller
// waits at prio. The caller holds mtx.
func (m *PrioMutex) classLocked(prio int) *prioClass {
	// Find the class of prio, or the position that keeps classes ascending.
	i, found := slices.BinarySearchFunc(m.classes, prio, func(c *prioClass, prio int) int {
		return cmp.Compare(c.prio, prio)
	})
	if found {
		return m.classes[i]
	}

	// Add the class, and publish it as top when it is the highest.
	class := &prioClass{prio: prio}
	m.classes = slices.Insert(m.classes, i, class)
	if i == len(m.classes)-1 {
		m.top.Store(class)
	}
	return class
}

// removeClassLocked removes a class whose queue emptied. The caller holds
// mtx.
func (m *PrioMutex) removeClassLocked(class *prioClass) {
	m.classes = slices.DeleteFunc(m.classes, func(c *prioClass) bool {
		return c == class
	})
	m.storeTopLocked()
}

// storeTopLocked publishes the last class as top. The caller holds mtx.
func (m *PrioMutex) storeTopLocked() {
	if len(m.classes) == 0 {
		m.classes = nil
		m.top.Store(nil)
		return
	}
	m.top.Store(m.classes[len(m.classes)-1])
}

// unlockLocked hands the lock to the oldest waiter of the highest priority,
// or unlocks the mutex when none waits. The caller holds mtx.
func (m *PrioMutex) unlockLocked() {
	// Unlock when no caller waits.
	if len(m.classes) == 0 {
		m.locked = false
		return
	}

	// Grant the lock to the head of the highest class, dropping the class
	// once it empties.
	class := m.classes[len(m.classes)-1]
	next := class.waiters[0]
	class.waiters[0] = nil
	class.waiters = class.waiters[1:]
	if len(class.waiters) == 0 {
		m.classes[len(m.classes)-1] = nil
		m.classes = m.classes[:len(m.classes)-1]
		m.storeTopLocked()
	}
	close(next)
}
