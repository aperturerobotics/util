//go:build windows

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// ErrorLockViolation is the error code returned from Windows when a lock would block.
const ErrorLockViolation windows.Errno = 0x21 // 33

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
		fh, err := os.OpenFile(f.path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return false, err
		}
		f.fh = fh
	}

	err := windows.LockFileEx(
		windows.Handle(f.fh.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&windows.Overlapped{},
	)
	if err != nil {
		if errors.Is(err, ErrorLockViolation) || errors.Is(err, windows.ERROR_IO_PENDING) {
			return false, nil
		}
		if !errors.Is(err, windows.Errno(0)) {
			return false, err
		}
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

	err := windows.UnlockFileEx(
		windows.Handle(f.fh.Fd()),
		0,
		1,
		0,
		&windows.Overlapped{},
	)
	if err != nil && !errors.Is(err, windows.Errno(0)) {
		return err
	}

	f.held = false
	_ = f.fh.Close()
	f.fh = nil

	return nil
}
