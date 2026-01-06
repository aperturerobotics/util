// Package flock implements a cross-platform file lock.
package flock

import (
	"context"
	"os"
	"sync"
	"time"
)

// Flock is a cross-platform file lock.
// The zero value is not usable; use New to create a Flock.
type Flock struct {
	path string
	mu   sync.Mutex
	fh   *os.File
	held bool
}

// New creates a new Flock for the given path.
// The lock file will be created if it does not exist.
func New(path string) *Flock {
	return &Flock{path: path}
}

// Path returns the path to the lock file.
func (f *Flock) Path() string {
	return f.path
}

// Locked returns whether this Flock instance holds the lock.
// Note: by the time you use the returned value, the state may have changed.
func (f *Flock) Locked() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.held
}

// Lock acquires an exclusive lock, blocking until available or context is cancelled.
// Uses exponential backoff starting at 50ms up to 200ms between retries.
func (f *Flock) Lock(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	const minDelay = 50 * time.Millisecond
	const maxDelay = 200 * time.Millisecond
	delay := minDelay

	for {
		locked, err := f.TryLock()
		if err != nil {
			return err
		}
		if locked {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Exponential backoff
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
