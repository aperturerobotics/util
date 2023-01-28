package conc

import (
	"context"
	"sync"

	"github.com/aperturerobotics/util/broadcast"
	"github.com/aperturerobotics/util/linkedlist"
)

// ConcurrentQueue is a pool of goroutines processing a stream of jobs.
// Job callbacks are called in the order they are added.
type ConcurrentQueue struct {
	// mtx guards below fields
	mtx sync.Mutex
	// bcast is broadcasted when fields change
	bcast broadcast.Broadcast
	// maxConcurrency is the concurrency limit or 0 if none
	maxConcurrency int
	// running is the number of running goroutines.
	running int
	// jobQueue is the job queue linked list.
	jobQueue *linkedlist.LinkedList[func()]
}

// NewConcurrentQueue constructs a new stream concurrency manager.
// initialElems contains the initial set of queued entries.
// if maxConcurrency <= 0, spawns infinite goroutines.
func NewConcurrentQueue(maxConcurrency int, initialElems ...func()) *ConcurrentQueue {
	str := &ConcurrentQueue{
		jobQueue:       linkedlist.NewLinkedList(initialElems...),
		maxConcurrency: maxConcurrency,
	}
	if len(initialElems) != 0 {
		str.mtx.Lock()
		str.updateLocked()
		str.mtx.Unlock()
	}
	return str
}

// Enqueue enqueues a job callback to the stream.
// Returns the current number of queued jobs.
// Note: may return 0 if the job was started immediately & the queue is empty.
func (s *ConcurrentQueue) Enqueue(jobs ...func()) {
	if len(jobs) == 0 {
		return
	}
	s.mtx.Lock()
	for _, job := range jobs {
		if s.maxConcurrency <= 0 || s.running < s.maxConcurrency {
			s.running++
			go s.executeJob(job)
		} else {
			s.jobQueue.Push(job)
		}
	}
	s.bcast.Broadcast()
	s.mtx.Unlock()
}

// WaitIdle waits for no jobs to be running.
// Returns context.Canceled if ctx is canceled.
// errCh is an optional error channel.
func (s *ConcurrentQueue) WaitIdle(ctx context.Context, errCh <-chan error) error {
	var wait <-chan struct{}
	for {
		s.mtx.Lock()
		idle := s.running == 0 && s.jobQueue.IsEmpty()
		if !idle {
			wait = s.bcast.GetWaitCh()
		}
		s.mtx.Unlock()
		if idle {
			return nil
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-wait:
		}
	}
}

// updateLocked checks if we need to spawn any new routines.
// caller must hold mtx
func (s *ConcurrentQueue) updateLocked() {
	var dirty bool
	for s.maxConcurrency <= 0 || s.running < s.maxConcurrency {
		job, jobOk := s.jobQueue.Pop()
		if !jobOk {
			break
		}
		s.running++
		dirty = true
		go s.executeJob(job)
	}
	if dirty {
		s.bcast.Broadcast()
	}
}

// executeJob is a goroutine to execute a job function.
// will continue to run until there are no more jobs.
func (s *ConcurrentQueue) executeJob(job func()) {
	for {
		if job != nil {
			job()
		}
		s.mtx.Lock()
		var jobOk bool
		job, jobOk = s.jobQueue.Pop()
		if !jobOk {
			s.running--
			s.bcast.Broadcast()
		}
		s.mtx.Unlock()
		if !jobOk {
			return
		}
	}
}
