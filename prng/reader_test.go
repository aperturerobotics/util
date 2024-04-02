package prng

import (
	"slices"
	"testing"
)

func TestSourceToReader(t *testing.T) {
	src := BuildSeededRand([]byte("test source to reader"))
	reader := SourceToReader(src)

	// Define test cases with different buffer sizes
	expected := [][]byte{
		[]byte{0xbc, 0xbb, 0x88},
		[]byte{0x77, 0x8c, 0x93, 0x40, 0xb5},
		[]byte{0xda, 0xe6, 0x94, 0x95, 0xeb, 0xa2, 0x26},
		[]byte{0x51, 0xe4},
		[]byte{0xe6, 0x73, 0xdd, 0x4, 0x86, 0x83},
	}

	// Perform reads for each test case
	for _, tc := range expected {
		buf := make([]byte, len(tc))

		// Read data from the reader into the buffer
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("Read() error = %v, wantErr %v", err, false)
		}
		if n != len(buf) {
			t.Errorf("Read() got = %v, want %v", n, len(buf))
		}

		// Ensure we have consistent output
		if !slices.Equal(buf, tc) {
			t.Logf("bufSize: %d", len(tc))
			t.Logf("expected: %#v", expected)
			t.Logf("actual: %#v", buf)
			t.FailNow()
		}
	}
}
