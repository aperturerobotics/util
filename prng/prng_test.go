package prng

import (
	"slices"
	"testing"
)

// TestBuildSeededRand tests builds a random source seeded by data.
func TestBuildSeededRand(t *testing.T) {
	rnd := BuildSeededRand([]byte("testing TestBuildSeededRand"))

	// ensure we have consistent output
	expected := []uint64{0x4de92abdfe46af09, 0xb04f7dea4c9c5140, 0x65b78a432144035, 0x51b965a601c7f14c, 0x13c7f4665c1d62ba, 0x1a7e6af46e8e425b, 0xaa95b5b840cba2a0, 0xdd6b1e4ad2892ec7, 0xefbf289c56df8240, 0x416d2dccc09152b, 0x27904db0d3577b93, 0xd724a9b5091763f6, 0x85fb48d7f028ffa5, 0x11718cafa1f0b20f, 0x2ae682d75aed60e, 0x5903f7b00356422d, 0x3757c473e500a6f1, 0x2f52b5bc048442b, 0xbc0e63652c90911b, 0x9e13b433398ae24d, 0x4a97a1d755bb88d6, 0x1d07ef9fdc565b9d}
	nums := make([]uint64, len(expected))

	for i := range nums {
		nums[i] = rnd.Uint64()
	}

	if !slices.Equal(nums, expected) {
		t.Logf("expected: %#v", expected)
		t.Logf("actual: %#v", nums)
		t.FailNow()
	}
}
