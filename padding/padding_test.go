package padding

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestPadUnpad checks that every length round-trips, including empty data.
func TestPadUnpad(t *testing.T) {
	for size := 0; size <= 70; size++ {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			t.Fatal(err.Error())
		}

		// Pad a copy and check the padded length.
		og := bytes.Clone(data)
		padded := PadInPlace(data)
		if len(padded) == 0 || len(padded)%alignPaddingTo != 0 || len(padded) <= size {
			t.Fatalf("size %d: padded to %d bytes", size, len(padded))
		}

		// Unpad and compare with the original.
		unpadded, err := UnpadInPlace(padded)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if !bytes.Equal(unpadded, og) {
			t.Fatalf("size %d: pad unpad fail: %v != %v", size, unpadded, og)
		}
	}

	// Empty input has no trailer to read.
	if _, err := UnpadInPlace(nil); err == nil {
		t.Fatal("expected an error unpadding empty data")
	}
}
