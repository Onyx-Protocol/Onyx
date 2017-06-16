package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestExcessCommitment(t *testing.T) {
	q := ecmath.Scalar{17}
	msg := []byte("message")
	qc := CreateExcessCommitment(q, msg)
	if !qc.Validate() {
		t.Error("failed to validate excess commitment")
	}
	qc.msg = msg[1:]
	if qc.Validate() {
		t.Error("validated invalid excess commitment")
	}
}
