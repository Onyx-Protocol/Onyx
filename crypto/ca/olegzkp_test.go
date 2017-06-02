package ca

import (
	"chain/crypto/ed25519/ecmath"
	"encoding/hex"
	"testing"
)

func TestOlegZKP(t *testing.T) {
	msg := []byte("message")

	// l == 6
	xbytes := []string{
		"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
		"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c1",
		"483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b47aca3d80a58aa0f",
		"9a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c523",
		"1a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa1",
		"47aca3d80a58aa0f483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b",
	}
	x := make([]ecmath.Scalar, len(xbytes))
	for i, xb := range xbytes {
		hex.Decode(x[i][:], []byte(xb))
	}

	coeffBytes := [][]string{
		[]string{
			"044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da",
			"c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166",
		},
		[]string{
			"da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419",
			"66c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e0471",
		},
		[]string{
			"19da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c714",
			"7166c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e04",
		},
	}

	n := len(coeffBytes)
	m := len(coeffBytes[0])

	F := make([][]ecmath.Point, n)
	f := make([][]OlegZKPFunc, n)

	for i := 0; i < n; i++ {
		F[i] = make([]ecmath.Point, m)
		f[i] = make([]OlegZKPFunc, m)
		for j := 0; j < m; j++ {
			var coeff ecmath.Scalar
			hex.Decode(coeff[:], []byte(coeffBytes[i][j]))
			F[i][j].ScMulBase(&coeff)
			Fij := F[i][j]
			f[i][j] = func(v []ecmath.Scalar) ecmath.Point {
				result := Fij
				if i != 0 {
					result.ScMul(&result, &v[0])
				}
				return result
			}
		}
	}

	ozkp := CreateOlegZKP(msg, x, f, F, 0)
	if !ozkp.Validate(msg, f, F) {
		t.Error("failed to validate Oleg-ZKP")
	}
}
