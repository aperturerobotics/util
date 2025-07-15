package util_bufio

// SplitOnNul is a bufio.SplitFunc that splits on NUL characters.
func SplitOnNul(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := range data {
		if data[i] == '\x00' {
			return i + 1, data[:i], nil
		}
	}
	return 0, nil, nil
}
