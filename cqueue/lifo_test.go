package cqueue

import (
	"testing"
)

func TestAtomicLIFO_PushPopSingleGoroutine(t *testing.T) {
	lifo := &AtomicLIFO[int]{}

	lifo.Push(1)
	lifo.Push(2)
	lifo.Push(3)

	if v := lifo.Pop(); v != 3 {
		t.Fatalf("expected 3, got %v", v)
	}
	if v := lifo.Pop(); v != 2 {
		t.Fatalf("expected 2, got %v", v)
	}
	if v := lifo.Pop(); v != 1 {
		t.Fatalf("expected 1, got %v", v)
	}
	if v := lifo.Pop(); v != 0 {
		t.Fatalf("expected nil, got %v", v)
	}
}

func TestAtomicLIFO_ConsecutivePushesAndPops(t *testing.T) {
	lifo := &AtomicLIFO[int]{}
	const count = 1000

	for i := 0; i < count; i++ {
		lifo.Push(i)
	}

	for i := count - 1; i >= 0; i-- {
		if v := lifo.Pop(); v != i {
			t.Fatalf("expected %d, got %v", i, v)
		}
	}

	if v := lifo.Pop(); v != 0 {
		t.Fatalf("expected nil, got %v", v)
	}
}
