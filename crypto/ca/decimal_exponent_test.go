package ca

import "testing"

func TestDecimalExponents(t *testing.T) {
	// Test that the value of tenInv is really the inverse of 10 mod l
	res := multiplyAndAddScalars(ten, tenInv, ZeroScalar)
	if res != one {
		t.Errorf("got %x, want %x", res[:], one[:])
	}

	// Test that the stored values are inverses of each other
	for i := 1; i <= 20; i++ {
		res := multiplyAndAddScalars(powersOf10[i], powersOf10[-i], ZeroScalar)
		if res != one {
			t.Errorf("got %x, want %x", res[:], one[:])
		}
	}

	for i := -20; i < 20; i++ {
		res := multiplyAndAddScalars(powersOf10[i], ten, ZeroScalar)
		if res != powersOf10[i+1] {
			want := powersOf10[i+1]
			t.Errorf("got %x, want %x", res[:], want[:])
		}
	}

	for i := -20; i < 20; i++ {
		res := multiplyAndAddScalars(powersOf10[i+1], tenInv, ZeroScalar)
		if res != powersOf10[i] {
			want := powersOf10[i]
			t.Errorf("got %x, want %x", res[:], want[:])
		}
	}
}
