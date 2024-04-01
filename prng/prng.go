package prng

import (
	"crypto/sha256"
	"math/rand/v2"
)

// BuildSeededRand builds a random source seeded by data.
func BuildSeededRand(datas ...[]byte) rand.Source {
	h := sha256.New()
	_, _ = h.Write([]byte("prng seed random in BuildSeededRand"))
	for _, d := range datas {
		_, _ = h.Write(d)
	}
	sum := h.Sum(nil)
	var seed [32]byte
	copy(seed[:], sum)
	return rand.NewChaCha8(seed)
}
