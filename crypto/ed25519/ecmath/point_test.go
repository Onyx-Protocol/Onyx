package ecmath

import "testing"

// base is the ed25519 base point
var base Point

func init() {
	base.ScMulBase(&One)
}

func TestBasePointArith(t *testing.T) {
	var base1 Point
	base1.ScMul(&base, &One)
	if !base.ConstTimeEqual(&base1) {
		ebase := base.Encode()
		ebase1 := base1.Encode()
		t.Errorf("base [%x] != 1*base [%x]", ebase[:], ebase1[:])
	}

	Two := One
	Two.Add(&Two, &One)

	base2a := base
	base2a.Add(&base2a, &base)

	base2b := base
	base2b.ScMul(&base2b, &Two)

	if !base2a.ConstTimeEqual(&base2b) {
		ebase2a := base2a.Encode()
		ebase2b := base2b.Encode()
		t.Errorf("base+base [%x] != 2*base [%x] (1)", ebase2a[:], ebase2b[:])
	}

	var base2c Point
	base2c.ScMulBase(&Two)

	if !base2a.ConstTimeEqual(&base2c) {
		ebase2a := base2a.Encode()
		ebase2c := base2c.Encode()
		t.Errorf("base+base [%x] != 2*base [%x] (2)", ebase2a[:], ebase2c[:])
	}
}
