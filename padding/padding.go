// Package padding pads data to a multiple of 32 bytes.
//
// PadInPlace zero-fills the data to one byte short of the next multiple and
// appends a trailer byte holding the padding length, reusing the slice's
// capacity when it can. UnpadInPlace reads the trailer and removes the
// padding, rejecting a trailer that the data cannot hold.
package padding

import (
	"github.com/pkg/errors"
)

const alignPaddingTo = 32

// PadInPlace attempts to extend data out to 32 byte intervals.
// Appends a 1-byte trailer with the padding length.
func PadInPlace(data []byte) []byte {
	var paddingLen byte
	dataLen := len(data) + 1 // for extra padding length byte
	if dlm := dataLen % alignPaddingTo; dlm != 0 {
		paddingLen = byte(alignPaddingTo - dlm)
	}
	nlen := dataLen + int(paddingLen)
	if cap(data) >= nlen {
		oldLen := len(data)
		data = data[:nlen] // extend slice with existing capacity
		for i := oldLen; i < nlen; i++ {
			data[i] = 0 // zero out old region
		}
		data[len(data)-1] = paddingLen
	} else {
		og := data
		data = make([]byte, nlen) // zeroed by golang
		copy(data, og)
		data[len(data)-1] = paddingLen
		// original buffer is released
	}
	return data
}

// UnpadInPlace removes padding according to the appended length byte.
func UnpadInPlace(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("padded data is empty")
	}

	// The trailer byte and the padding it counts must fit in the data.
	paddingLen := int(data[len(data)-1])
	if paddingLen > len(data)-1 || paddingLen >= alignPaddingTo {
		return nil, errors.Errorf(
			"%d padding indicated but message is %d bytes",
			paddingLen,
			len(data),
		)
	}
	data = data[:len(data)-paddingLen-1]
	return data, nil
}
