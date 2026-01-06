//go:build !js && !plan9 && !wasip1

package flock

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f := New(path)

	if f == nil {
		t.Fatal("New returned nil")
	}
	if f.Path() != path {
		t.Errorf("Path() = %q, want %q", f.Path(), path)
	}
	if f.Locked() {
		t.Error("new Flock should not be locked")
	}
}

func TestTryLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f := New(path)
	defer f.Unlock()

	// First TryLock should succeed
	locked, err := f.TryLock()
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Error("TryLock() = false, want true")
	}
	if !f.Locked() {
		t.Error("Locked() = false after TryLock succeeded")
	}

	// Second TryLock on same instance should return true (already locked)
	locked, err = f.TryLock()
	if err != nil {
		t.Fatalf("second TryLock() error = %v", err)
	}
	if !locked {
		t.Error("second TryLock() = false, want true")
	}

	// TryLock from different Flock instance on same path should return false
	f2 := New(path)
	defer f2.Unlock()

	locked, err = f2.TryLock()
	if err != nil {
		t.Fatalf("f2.TryLock() error = %v", err)
	}
	if locked {
		t.Error("f2.TryLock() = true, want false (lock held by f)")
	}
}

func TestUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f := New(path)

	// Unlock on unlocked Flock should be safe
	if err := f.Unlock(); err != nil {
		t.Fatalf("Unlock() on unlocked Flock error = %v", err)
	}

	// Acquire lock
	locked, err := f.TryLock()
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Fatal("TryLock() = false, want true")
	}

	// Unlock should release
	if err := f.Unlock(); err != nil {
		t.Fatalf("Unlock() error = %v", err)
	}
	if f.Locked() {
		t.Error("Locked() = true after Unlock")
	}

	// Multiple Unlock calls should be safe
	if err := f.Unlock(); err != nil {
		t.Fatalf("second Unlock() error = %v", err)
	}

	// Another Flock should now be able to acquire the lock
	f2 := New(path)
	defer f2.Unlock()

	locked, err = f2.TryLock()
	if err != nil {
		t.Fatalf("f2.TryLock() after unlock error = %v", err)
	}
	if !locked {
		t.Error("f2.TryLock() = false after f.Unlock(), want true")
	}
}

func TestLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f := New(path)
	defer f.Unlock()

	ctx := context.Background()

	// Lock should succeed
	if err := f.Lock(ctx); err != nil {
		t.Fatalf("Lock() error = %v", err)
	}
	if !f.Locked() {
		t.Error("Locked() = false after Lock succeeded")
	}

	// Lock on same instance should succeed (already locked)
	if err := f.Lock(ctx); err != nil {
		t.Fatalf("second Lock() error = %v", err)
	}

	// Test blocking behavior: Lock from another Flock should block until released
	f2 := New(path)
	defer f2.Unlock()

	acquired := make(chan struct{})
	go func() {
		if err := f2.Lock(ctx); err != nil {
			t.Errorf("f2.Lock() error = %v", err)
		}
		close(acquired)
	}()

	// Give goroutine time to start and block
	select {
	case <-acquired:
		t.Fatal("f2.Lock() should block while f holds lock")
	case <-time.After(100 * time.Millisecond):
		// Expected: f2 is blocked
	}

	// Release lock from f
	if err := f.Unlock(); err != nil {
		t.Fatalf("f.Unlock() error = %v", err)
	}

	// f2 should now acquire
	select {
	case <-acquired:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("f2.Lock() did not acquire lock after f.Unlock()")
	}

	if !f2.Locked() {
		t.Error("f2.Locked() = false after Lock succeeded")
	}
}

func TestLockContextCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	// Hold lock with first Flock
	f := New(path)
	defer f.Unlock()

	locked, err := f.TryLock()
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Fatal("TryLock() = false, want true")
	}

	// Try to Lock with already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f2 := New(path)
	defer f2.Unlock()

	err = f2.Lock(ctx)
	if err != context.Canceled {
		t.Errorf("Lock() with cancelled context error = %v, want %v", err, context.Canceled)
	}
	if f2.Locked() {
		t.Error("f2.Locked() = true, want false")
	}
}

func TestLockContextTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	// Hold lock with first Flock
	f := New(path)
	defer f.Unlock()

	locked, err := f.TryLock()
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Fatal("TryLock() = false, want true")
	}

	// Try to Lock with short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	f2 := New(path)
	defer f2.Unlock()

	err = f2.Lock(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Lock() with timeout context error = %v, want %v", err, context.DeadlineExceeded)
	}
	if f2.Locked() {
		t.Error("f2.Locked() = true, want false")
	}
}
