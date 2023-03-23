package util_bufio

// SplitOnNul is a bufio.SplitFunc that splits on NUL characters.
func SplitOnNul(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '\x00' {
			return i + 1, data[:i], nil
		}
	}
	return 0, nil, nil
}
