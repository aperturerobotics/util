//go:build (!unix && !windows) || plan9

package flock

import "errors"

// ErrUnsupported is returned on platforms that don't support file locking.
var ErrUnsupported = errors.New("flock: unsupported platform")

// TryLock attempts to acquire an exclusive lock.
// Returns ErrUnsupported on this platform.
func (f *Flock) TryLock() (bool, error) {
	return false, ErrUnsupported
}

// Unlock releases the lock.
// Returns ErrUnsupported on this platform.
func (f *Flock) Unlock() error {
	return ErrUnsupported
}
