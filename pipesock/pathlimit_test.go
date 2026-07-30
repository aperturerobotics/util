//go:build !windows

package pipesock

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// longPipeRoot builds a directory whose path exceeds any sun_path limit both
// as an absolute path and as a path relative to the working directory.
func longPipeRoot(t *testing.T) string {
	t.Helper()
	base, err := os.MkdirTemp("/tmp", "ps")
	if err != nil {
		t.Fatalf("make temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	root := base
	for range 8 {
		root = filepath.Join(root, strings.Repeat("d", 24))
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("make nested dir: %v", err)
	}
	return root
}

func TestBuildPipeListenerRejectsOverLongPath(t *testing.T) {
	root := longPipeRoot(t)
	listener, err := BuildPipeListener(newTestLogger(), root, "startup")
	if err == nil {
		_ = listener.Close()
		t.Fatal("expected an error for a socket path longer than the platform limit")
	}
	if !strings.Contains(err.Error(), "binds at most") {
		t.Fatalf("error does not name the platform limit: %v", err)
	}
	if !strings.Contains(err.Error(), root) {
		t.Fatalf("error does not name the offending path: %v", err)
	}
}

func TestDialPipeListenerRejectsOverLongPath(t *testing.T) {
	root := longPipeRoot(t)
	conn, err := DialPipeListener(context.Background(), newTestLogger(), root, "startup")
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected an error for a socket path longer than the platform limit")
	}
	if !strings.Contains(err.Error(), "binds at most") {
		t.Fatalf("error does not name the platform limit: %v", err)
	}
}

// TestBuildPipeListenerUsesRelativePathUnderTheLimit covers the case the
// shortening exists for: an absolute path the platform refuses, reachable by a
// relative path it accepts.
func TestBuildPipeListenerUsesRelativePathUnderTheLimit(t *testing.T) {
	root := longPipeRoot(t)
	if len(root) <= maxPipePathLen {
		t.Fatalf("test root is not long enough to exercise the shortening: %d", len(root))
	}
	t.Chdir(filepath.Dir(root))

	listener, err := BuildPipeListener(newTestLogger(), root, "startup")
	if err != nil {
		t.Fatalf("listen through the relative path: %v", err)
	}
	defer listener.Close()

	if got := listener.Addr().String(); filepath.IsAbs(got) {
		t.Fatalf("listener bound the absolute path rather than the shorter relative one: %s", got)
	}
}

func TestMaxPipePathLenMatchesPlatform(t *testing.T) {
	if maxPipePathLen < 90 || maxPipePathLen > 120 {
		t.Fatalf("sun_path limit outside the range every unix platform uses: %d", maxPipePathLen)
	}
}
