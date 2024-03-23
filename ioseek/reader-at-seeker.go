package ioseek

import (
	"errors"
	"io"
)

// ReaderAtSeeker wraps an io.ReaderAt to provide io.Seeker behavior.
// It embeds io.RederAt, thereby inheriting its methods directly.
type ReaderAtSeeker struct {
	io.ReaderAt
	size   int64 // The size of the file
	offset int64 // The current offset within the file
}

// NewReaderAtSeeker creates a new ReaderAtSeeker with the provided io.ReaderAt and file size.
func NewReaderAtSeeker(readerAt io.ReaderAt, size int64) *ReaderAtSeeker {
	return &ReaderAtSeeker{
		ReaderAt: readerAt,
		size:     size,
		offset:   0,
	}
}

// Seek implements the io.Seeker interface.
func (r *ReaderAtSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.size + offset
	default:
		return 0, errors.New("ReaderAtSeeker.Seek: invalid whence")
	}

	// Check for seeking before the start of the file
	if newOffset < 0 {
		return 0, errors.New("ReaderAtSeeker.Seek: negative position")
	}

	// If seeking beyond the end of the file, return 0 and io.EOF
	if newOffset > r.size {
		return 0, io.EOF
	}

	r.offset = newOffset
	return r.offset, nil
}

// Read reads up to len(p) bytes into p.
func (r *ReaderAtSeeker) Read(p []byte) (n int, err error) {
	n, err = r.ReaderAt.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// _ is a type assertion
var _ io.ReadSeeker = ((*ReaderAtSeeker)(nil))
