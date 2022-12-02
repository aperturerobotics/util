package iosizer

import (
	"io"
	"sync/atomic"
)

// SizeReadWriter implements io methods keeping total size metrics.
type SizeReadWriter struct {
	total uint64
	rdr   io.Reader
	wtr   io.Writer
}

// NewSizeReadWriter constructs a read/writer with size metrics.
func NewSizeReadWriter(rdr io.Reader, writer io.Writer) *SizeReadWriter {
	return &SizeReadWriter{rdr: rdr, wtr: writer}
}

// TotalSize returns the total amount of data transferred.
func (s *SizeReadWriter) TotalSize() uint64 {
	return atomic.LoadUint64(&s.total)
}

// Read reads data from the source.
func (s *SizeReadWriter) Read(p []byte) (n int, err error) {
	if s.rdr == nil {
		return 0, io.EOF
	}
	n, err = s.rdr.Read(p)
	if n != 0 {
		atomic.AddUint64(&s.total, uint64(n))
	}
	return
}

// Write writes data to the writer.
func (s *SizeReadWriter) Write(p []byte) (n int, err error) {
	if s.wtr == nil {
		return 0, io.EOF
	}
	n, err = s.wtr.Write(p)
	if n != 0 {
		atomic.AddUint64(&s.total, uint64(n))
	}
	return
}

// _ is a type assertion
var _ io.ReadWriter = ((*SizeReadWriter)(nil))
