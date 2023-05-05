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
	q := NewConcurrentQueue(2, mkJob())
	queued, running := q.Enqueue(mkJob(), mkJob(), mkJob(), mkJob())
	if queued != 3 || running != 2 {
		t.FailNow()
	}

	// expect 0 + 1 to complete immediately
	<-jobs[0]
	<-jobs[1]

	// expect 2 + 3 + 4 to not be started yet
	for i := 2; i <= 4; i++ {
		select {
		case <-jobs[i]:
			t.Fail()
		default:
		}
	}

	close(complete)

	// expect 2 + 3 + 4 to complete
	<-jobs[2]
	<-jobs[3]
	<-jobs[4]
}
