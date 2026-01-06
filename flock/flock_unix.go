//go:build unix

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// TryLock attempts to acquire an exclusive lock.
// Returns true if the lock was acquired, false if it is held by another process.
// Returns an error if the lock could not be attempted.
func (f *Flock) TryLock() (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.held {
		return true, nil
	}

	if f.fh == nil {
		fh, err := os.OpenFile(f.path, os.O_CREATE|os.O_RDONLY, 0o600)
		if err != nil {
			return false, err
		}
		f.fh = fh
	}

	err := unix.Flock(int(f.fh.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return false, nil
		}
		return false, err
	}

	f.held = true
	return true, nil
}

// Unlock releases the lock.
// Safe to call multiple times.
func (f *Flock) Unlock() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.held || f.fh == nil {
		return nil
	}

	err := unix.Flock(int(f.fh.Fd()), unix.LOCK_UN)
	if err != nil {
		return err
	}

	f.held = false
	_ = f.fh.Close()
	f.fh = nil

	return nil
}
