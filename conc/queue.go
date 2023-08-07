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
	// jobQueueSize is the current size of jobQueue
	jobQueueSize int
}

// NewConcurrentQueue constructs a new stream concurrency manager.
// initialElems contains the initial set of queued entries.
// if maxConcurrency <= 0, spawns infinite goroutines.
func NewConcurrentQueue(maxConcurrency int, initialElems ...func()) *ConcurrentQueue {
	str := &ConcurrentQueue{
		jobQueue:       linkedlist.NewLinkedList(initialElems...),
		jobQueueSize:   len(initialElems),
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
// If possible, the job is started immediately and skips the queue.
// Returns the current number of queued and running jobs.
func (s *ConcurrentQueue) Enqueue(jobs ...func()) (queued, running int) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if len(jobs) == 0 {
		return s.jobQueueSize, s.running
	}

	for _, job := range jobs {
		if s.maxConcurrency <= 0 || s.running < s.maxConcurrency {
			s.running++
			go s.executeJob(job)
		} else {
			s.jobQueueSize++
			s.jobQueue.Push(job)
		}
	}

	s.bcast.Broadcast()
	return s.jobQueueSize, s.running
}

// WaitIdle waits for no jobs to be running.
// Returns context.Canceled if ctx is canceled.
// errCh is an optional error channel.
func (s *ConcurrentQueue) WaitIdle(ctx context.Context, errCh <-chan error) error {
	var wait <-chan struct{}
	for {
		s.mtx.Lock()
		idle := s.running == 0 && s.jobQueueSize == 0
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

// WatchState watches the concurrent queue state.
// If the callback returns an error or false, returns that error or nil.
// Returns nil immediately if callback is nil.
// Returns context.Canceled if ctx is canceled.
// errCh is an optional error channel.
func (s *ConcurrentQueue) WatchState(
	ctx context.Context,
	errCh <-chan error,
	cb func(queued, running int) (bool, error),
) error {
	if cb == nil {
		return nil
	}

	for {
		s.mtx.Lock()
		queued, running := s.jobQueueSize, s.running
		waitCh := s.bcast.GetWaitCh()
		s.mtx.Unlock()

		cntu, err := cb(queued, running)
		if err != nil || !cntu {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-waitCh:
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
		s.jobQueueSize--
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
		} else {
			s.jobQueueSize--
		}
		s.mtx.Unlock()
		if !jobOk {
			return
		}
	}
}
