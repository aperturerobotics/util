package ioproxy

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
)

// TestProxyStreams checks that bytes written to one stream's far end arrive at
// the other stream's far end, and that closing one side closes both.
func TestProxyStreams(t *testing.T) {
	// Each proxied stream is one end of its own pipe; the test holds the far ends.
	data := []byte("Hello, World!")
	s1, far1 := net.Pipe()
	s2, far2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(2)
	ProxyStreams(s1, s2, wg.Done)

	// Write through the first stream and close it.
	go func() {
		_, _ = far1.Write(data)
		_ = far1.Close()
	}()

	// Read until the proxy closes the second stream.
	buf, err := io.ReadAll(far2)
	if err != nil {
		t.Fatal(err.Error())
	}
	wg.Wait()

	// Both directions stopped and the bytes arrived unchanged.
	if !bytes.Equal(data, buf) {
		t.Fatalf("proxied %q, want %q", buf, data)
	}
}
