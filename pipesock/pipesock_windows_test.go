//go:build windows

package pipesock

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestPipeListener checks that clients reach the instance Accept hands out,
// which Wine breaks when the name is reserved by an instance without access.
func TestPipeListener(t *testing.T) {
	le := logrus.New().WithField("test", t.Name())
	le.Logger.SetOutput(io.Discard)
	root := t.TempDir()
	uuid := "test-" + strconv.FormatInt(time.Now().UnixNano(), 36)

	lis, err := BuildPipeListener(le, root, uuid)
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Each client dials before Accept runs, then both ends exchange a line.
	for i := range 3 {
		client, err := DialPipeListener(ctx, le, root, uuid)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		msg := []byte("hello " + strconv.Itoa(i))
		if _, err := client.Write(msg); err != nil {
			t.Fatalf("client write %d: %v", i, err)
		}

		server, err := lis.Accept()
		if err != nil {
			t.Fatalf("accept %d: %v", i, err)
		}
		if err := server.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatalf("set deadline %d: %v", i, err)
		}
		got := make([]byte, len(msg))
		if _, err := io.ReadFull(server, got); err != nil || string(got) != string(msg) {
			t.Fatalf("server read %d: %q, %v", i, got, err)
		}
		if _, err := server.Write(msg); err != nil {
			t.Fatalf("server write %d: %v", i, err)
		}
		if _, err := io.ReadFull(client, got); err != nil || string(got) != string(msg) {
			t.Fatalf("client read %d: %q, %v", i, got, err)
		}
		_ = client.Close()
		_ = server.Close()
	}

	// Close wakes an Accept that is waiting for a client.
	errCh := make(chan error, 1)
	go func() {
		_, err := lis.Accept()
		errCh <- err
	}()
	time.Sleep(100 * time.Millisecond)
	_ = lis.Close()
	select {
	case err := <-errCh:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("accept after close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("close did not wake accept")
	}
}
