package ioproxy

import (
	"io"
)

// ProxyStreams constructs read/write pumps to proxy two streams.
// if either stream is closed, the other will be closed.
// The two routines will exit when the streams are closed.
// Cb will be called if either of the streams close, and will be called twice.
func ProxyStreams(s1, s2 io.ReadWriteCloser, cb func()) {
	go proxyTo(s1, s2, cb)
	go proxyTo(s2, s1, cb)
}

func proxyTo(s1, s2 io.ReadWriteCloser, cb func()) {
	buf := make([]byte, 8192)
	_, _ = io.CopyBuffer(s2, s1, buf)
	// io.Copy(s2, s1)
	s1.Close()
	s2.Close()
	if cb != nil {
		cb()
	}
}
