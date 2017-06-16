package ca

import (
	"encoding/hex"
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestOlegZKP(t *testing.T) {
	msg := []byte("message")

	// l == 5
	xbytes := []string{
		"4f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da04",
		"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c1",
		"483262b2722ec4a6e967af08d0ed3d51f954e2b9cab2b51b47aca3d80a58aa0f",
		"9a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c523",
		"1a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa1",
	}

	// additional coefficients for F[i][j] where i>0
	coeffBytes := [][]string{
		[]string{
			"6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da044f",
			"e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c159",
		},
		[]string{
			"87e9e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da044f6d",
			"0a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c159e1",
		},
		[]string{
			"e83dc1dc6868c13fa1ab6af977bcedec0304c5239a87c7c71419da044f6d87e9",
			"37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e047166c159e10a1f",
		},
	}

	var (
		n = len(coeffBytes)
		m = len(coeffBytes[0])
		l = len(xbytes)

		G ecmath.Point

		x     = make([]ecmath.Scalar, l)
		A     = make([]ecmath.Point, l)
		f     = make([]OlegZKPFunc, m)
		F     = make([][]ecmath.Point, n)
		coeff = make([][]ecmath.Scalar, n)
	)

	G.ScMulBase(&ecmath.One)

	for i := 0; i < l; i++ {
		hex.Decode(x[i][:], []byte(xbytes[i]))
		if i == 0 {
			A[i] = G
		} else {
			A[i].Add(&A[i-1], &G) // A[n] == (n+1)*G
		}
	}
	for j := 0; j < m; j++ {
		f[j] = func(y []ecmath.Scalar) ecmath.Point {
			var result ecmath.Point
			result.ScMul(&A[0], &y[0])
			for k := 1; k < l; k++ {
				var P ecmath.Point
				P.ScMul(&A[k], &y[k])
				result.Add(&result, &P)
			}
			return result
		}
	}
	for i := 0; i < n; i++ {
		F[i] = make([]ecmath.Point, m)
		coeff[i] = make([]ecmath.Scalar, m)
		for j := 0; j < m; j++ {
			F[i][j] = f[j](x)
			if i > 0 {
				hex.Decode(coeff[i][j][:], []byte(coeffBytes[i][j]))
				F[i][j].ScMul(&F[i][j], &coeff[i][j])
			}
		}
	}
	ozkp := CreateOlegZKP(msg, x, f, F, 0)
	if !ozkp.Validate(msg, f, F) {
		t.Error("failed to validate Oleg-ZKP")
	}
	ozkp = CreateOlegZKP(msg[1:], x, f, F, 0)
	if ozkp.Validate(msg, f, F) {
		t.Error("validated invalid Oleg-ZKP")
	}
	ozkp = CreateOlegZKP(msg, append(x[1:], x[0]), f, F, 0)
	if ozkp.Validate(msg, f, F) {
		t.Error("validated invalid Oleg-ZKP")
	}
	ozkp = CreateOlegZKP(msg, x, f, append(F[1:], F[0]), 0)
	if ozkp.Validate(msg, f, F) {
		t.Error("validated invalid Oleg-ZKP")
	}
	ozkp = CreateOlegZKP(msg, x, f, F, 1)
	if ozkp.Validate(msg, f, F) {
		t.Error("validated invalid Oleg-ZKP")
	}
}
