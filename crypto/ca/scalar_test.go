package ca

import "testing"

func TestTopFourBitsAreZero(t *testing.T) {
	// Due to subgroup's order 2^252 + 27742317777372353535851937790883648493,
	// top 4 bits of every modulo-reduced scalar are zero mostly always, but not quite.
	// There's a 1 in 2^124 chance that 4th bit from the top will be 1.
	// This test shows (on a very limited dataset) that top 4 bits are zero for virtually all randomly selected scalars.
	for i := uint64(0); i <= 10000; i++ {
		ienc := uint64le(i)
		e := reducedScalar(hash512(ienc[:]))
		if (e[31] & 0xf0) != 0 {
			t.Fatalf("Top four bits are not zero for SHA3-512(%d) mod l: %x", i, e)
		}
	}
}
