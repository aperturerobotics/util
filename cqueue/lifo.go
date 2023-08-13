package cqueue

import (
	"sync/atomic"
)

// atomicLIFONode represents a single element in the LIFO.
type atomicLIFONode[T any] struct {
	value T
	next  *atomicLIFONode[T]
}

// AtomicLIFO implements an atomic last-in-first-out linked-list.
type AtomicLIFO[T any] struct {
	top atomic.Pointer[atomicLIFONode[T]]
}

// Push atomically adds a value to the top of the LIFO.
func (q *AtomicLIFO[T]) Push(value T) {
	newNode := &atomicLIFONode[T]{value: value}

	for {
		// Read the current top.
		oldTop := q.top.Load()

		// Set the next of the new atomicLIFONode to the current top.
		newNode.next = oldTop

		// Try to set the new atomicLIFONode as the new top.
		if q.top.CompareAndSwap(oldTop, newNode) {
			break
		}
	}
}

// Pop atomically removes and returns the top value of the LIFO.
// It returns the zero value (nil) if the LIFO is empty.
func (q *AtomicLIFO[T]) Pop() T {
	for {
		// Read the current top.
		oldTop := q.top.Load()
		if oldTop == nil {
			var empty T
			return empty
		}

		// Read the next atomicLIFONode after the top.
		next := oldTop.next

		// Try to set the next atomicLIFONode as the new top.
		if q.top.CompareAndSwap(oldTop, next) {
			return oldTop.value
		}
	}
}
