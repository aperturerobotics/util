package ioproxy

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

func TestProxyStreams(t *testing.T) {
	data := []byte("Hello, World!")

	// Create pipe connections to simulate the streams.
	s1Reader, s1Writer := io.Pipe()
	s2Reader, s2Writer := io.Pipe()

	// Wrap the pipe connections with our custom readWriteCloser type.
	s1 := &readWriteCloser{
		Reader: s1Reader,
		Writer: s1Writer,
		Closer: s1Writer,
	}
	s2 := &readWriteCloser{
		Reader: s2Reader,
		Writer: s2Writer,
		Closer: s2Writer,
	}

	// Initialize the wait group to synchronize the callbacks.
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Create a callback function to be called when the streams are closed.
	cb := func() {
		wg.Done()
	}

	// Start proxying the streams.
	ProxyStreams(s1, s2, cb)

	// Write the data to s1Writer.
	go func() {
		_, _ = s1Writer.Write(data)
		s1Writer.Close()
	}()

	// Read the data from s2Reader.
	var result []byte
	buf := make([]byte, 8192)
	for {
		n, err := s2Reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		result = append(result, buf[:n]...)
	}

	// Wait for the callbacks to be called.
	wg.Wait()

	// Check if the data was correctly proxied.
	if !bytes.Equal(data, result) {
		t.Fail()
	}
}

type readWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}
