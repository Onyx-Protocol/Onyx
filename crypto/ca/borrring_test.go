package ca

import (
	"encoding/hex"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519/ecmath"
)

func TestBorrRingSig(t *testing.T) {
	msg := []byte("message")

	var B0 ecmath.Point
	B0.ScMulBase(&ecmath.One)

	pbytes := [][]string{
		[]string{
			"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
			"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c1",
		},
		[]string{
			"9a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c523",
			"1a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa1",
		},
		[]string{
			"77bcedec0304c5239a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af9",
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736",
		},
	}

	n := len(pbytes)
	m := len(pbytes[0])

	B := make([][]ecmath.Point, n)
	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		B[i] = make([]ecmath.Point, 1)
		B[i][0] = B0
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			P[i][j] = make([]ecmath.Point, 1)
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))
			P[i][j][0].ScMul(&B0, &p2)
			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n)
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	t.Log(spew.Sdump(brs))
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}
