package scrub

// Scrub clears a buffer with zeros.
// Prevents reading sensitive data before memory is overwritten.
func Scrub(buf []byte) {
	// compiler optimizes this to memset
	for i := range buf {
		buf[i] = 0
	}
}
