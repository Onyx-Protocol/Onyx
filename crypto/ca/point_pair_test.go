package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestZeroPair(t *testing.T) {
	z1 := ZeroPointPair
	z2 := ZeroPointPair

	if !z1.Point1.ConstTimeEqual(&ecmath.ZeroPoint) ||
		!z1.Point2.ConstTimeEqual(&ecmath.ZeroPoint) {
		t.Errorf("zero point pair must be composed of two zero points")
	}

	if !z1.ConstTimeEqual(&z2) {
		t.Errorf("zero point pairs must be equal to each other")
	}

	var z3 PointPair
	z3.Add(&z1, &z2)

	if !z3.ConstTimeEqual(&z1) {
		t.Errorf("zero point pairs must be added to another zero point pair")
	}
}
