package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestExcessCommitment(t *testing.T) {
	q := ecmath.Scalar{17}
	msg := []byte("message")
	qc := CreateExcessCommitment(q, msg)
	if !qc.Validate(msg) {
		t.Error("failed to validate excess commitment")
	}
	if qc.Validate(msg[1:]) {
		t.Error("validated invalid excess commitment")
	}
}
