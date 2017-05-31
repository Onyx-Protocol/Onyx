package ca

import (
	"encoding/hex"
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestRingSig(t *testing.T) {
	msg := []byte("message")

	var B ecmath.Point
	B.ScMulBase(&ecmath.One)

	pbytes := []string{
		"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
		"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c1",
		"483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b47aca3d80a58aa0f",
	}

	var p ecmath.Scalar

	P := make([][]ecmath.Point, 3)
	for i := 0; i < 3; i++ {
		P[i] = make([]ecmath.Point, 1)
		var p2 ecmath.Scalar
		hex.Decode(p2[:], []byte(pbytes[i]))
		P[i][0].ScMul(&B, &p2)
		if i == 0 {
			p = p2
		}
	}
	P[0][0].ScMul(&B, &p)

	rs := CreateRingSignature(msg, []ecmath.Point{B}, P, 0, p)
	if !rs.Validate(msg, []ecmath.Point{B}, P) {
		t.Error("failed to validate ring signature")
	}
}
