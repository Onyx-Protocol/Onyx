package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestDecimalExponents(t *testing.T) {
	// Test that the value of tenInv is really the inverse of 10 mod l
	var res ecmath.Scalar
	res.Mul(&ten, &tenInv)
	if res != ecmath.One {
		t.Errorf("got %x, want %x", res[:], ecmath.One[:])
	}

	// Test that the stored values are inverses of each other
	for i := 1; i < 20; i++ {
		res.Mul(&powersOf10[i], &invPowersOf10[i])
		if res != ecmath.One {
			t.Errorf("got %x, want %x", res[:], ecmath.One[:])
		}
	}

	for i := -19; i < 19; i++ {
		res.Mul(powerOf10(i), &ten)
		want := powerOf10(i + 1)
		if res != *want {
			t.Errorf("got %x, want %x", res[:], want[:])
		}
	}

	for i := -19; i < 19; i++ {
		res.Mul(powerOf10(i+1), &tenInv)
		want := powerOf10(i)
		if res != *want {
			t.Errorf("got %x, want %x", res[:], want[:])
		}
	}
}
