package prng

import (
	"io"
	"math/rand/v2"
)

// randReader wraps a Source to implement io.Reader using random uint64 values.
type randReader struct {
	src rand.Source
	buf [8]byte
	off int
}

// SourceToReader builds an io.Reader from a rand.Source.
//
// NOTE: the reader is not safe for concurrent use.
func SourceToReader(src rand.Source) io.Reader {
	return &randReader{src: src}
}

// BuildSeededReader builds a random reader seeded by data.
//
// NOTE: the reader is not safe for concurrent use.
func BuildSeededReader(datas ...[]byte) io.Reader {
	rd := BuildSeededRand(datas...)
	return SourceToReader(rd)
}

// Read generates random data and writes it into p.
// It reads up to len(p) bytes into p and returns the number of bytes read and any error encountered.
func (r *randReader) Read(p []byte) (n int, err error) {
	for n < len(p) {
		if r.off == 0 {
			// Generate a new random uint64 value and store it in the buffer.
			val := r.src.Uint64()
			for i := range 8 {
				r.buf[i] = byte(val >> (i * 8))
			}
		}

		// Determine how many bytes to copy from the buffer.
		remaining := min(len(p)-n, 8-r.off)

		// Copy bytes from the buffer into p.
		copy(p[n:], r.buf[r.off:r.off+remaining])
		n += remaining
		r.off = (r.off + remaining) % 8
	}
	return n, nil
}
