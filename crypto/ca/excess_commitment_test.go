package ca

import "testing"

func TestExcessCommitment(t *testing.T) {
	lc := CreateExcessCommitment(Scalar{0})
	if !lc.Verify() {
		t.Errorf("not verified")
	}

	lc.e[0] ^= 1
	if lc.Verify() {
		t.Errorf("unexpected verification")
	}
	lc.e[0] ^= 1

	lc.s[0] ^= 1
	if lc.Verify() {
		t.Errorf("unexpected verification")
	}
	lc.s[0] ^= 1
}
