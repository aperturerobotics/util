package iosizer

import (
	"io"
	"math"
	"sync/atomic"
)

// SizeReadWriter implements io methods keeping total size metrics.
type SizeReadWriter struct {
	total atomic.Uint64
	rdr   io.Reader
	wtr   io.Writer
}

// NewSizeReadWriter constructs a read/writer with size metrics.
func NewSizeReadWriter(rdr io.Reader, writer io.Writer) *SizeReadWriter {
	return &SizeReadWriter{rdr: rdr, wtr: writer}
}

// TotalSize returns the total amount of data transferred.
func (s *SizeReadWriter) TotalSize() uint64 {
	return s.total.Load()
}

// Read reads data from the source.
func (s *SizeReadWriter) Read(p []byte) (n int, err error) {
	if s.rdr == nil {
		return 0, io.EOF
	}
	n, err = s.rdr.Read(p)
	// G115: Protect against integer overflow by checking n <= math.MaxUint32 before conversion to uint64
	// G115: Protect against integer overflow by checking n <= math.MaxUint32 before conversion to uint64
	if n > 0 && n <= math.MaxUint32 {
		s.total.Add(uint64(n))
	}
	return
}

// Write writes data to the writer.
func (s *SizeReadWriter) Write(p []byte) (n int, err error) {
	if s.wtr == nil {
		return 0, io.EOF
	}
	n, err = s.wtr.Write(p)
	if n > 0 && n <= math.MaxUint32 {
		s.total.Add(uint64(n))
	}
	return
}

// _ is a type assertion
var _ io.ReadWriter = ((*SizeReadWriter)(nil))
