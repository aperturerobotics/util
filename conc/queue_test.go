package conc

import (
	"testing"
)

// TestConcurrentQueue tests the concurrent queue type.
func TestConcurrentQueue(t *testing.T) {
	complete := make(chan struct{})
	jobs := make(map[int]chan struct{})
	n := 0
	mkJob := func() func() {
		doneCh := make(chan struct{})
		jobs[n] = doneCh
		n++
		return func() {
			close(doneCh)
			<-complete
		}
	}
	q := NewConcurrentQueue(2, mkJob(), mkJob())
	q.Enqueue(mkJob(), mkJob())

	// expect 0 + 1 to complete immediately
	<-jobs[0]
	<-jobs[1]

	// expect 2 + 3 to not be started yet
	select {
	case <-jobs[2]:
		t.Fail()
	default:
	}
	select {
	case <-jobs[3]:
		t.Fail()
	default:
	}

	close(complete)

	// expect 2 + 3 to complete
	<-jobs[2]
	<-jobs[3]
}
