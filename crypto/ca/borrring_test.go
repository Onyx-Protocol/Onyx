package ca

import (
	"encoding/hex"
	"testing"

	"chain/crypto/ed25519/ecmath"
)

// n=1 ring, m=1 sig, M=1 base point
func TestBorrRingSig111(t *testing.T) {
	msg := []byte("message")

	pbytes := [][]string{
		[]string{
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736",
		},
	}

	n := len(pbytes)
	m := len(pbytes[0])

	B := []ecmath.Point{G}
	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			P[i][j] = make([]ecmath.Point, 1)
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))
			P[i][j][0].ScMul(&G, &p2)
			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n) // n zeroes
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}

// n=1 ring, m=2 sigs, M=1 base point
func TestBorrRingSig121(t *testing.T) {
	msg := []byte("message")

	pbytes := [][]string{
		[]string{
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736",
			"7b0e047166c159e10f2f0e4a16c159e10a1f379f2d6d83736a1f37f2f0e4a1f2",
		},
	}

	n := len(pbytes)
	m := len(pbytes[0])

	B := []ecmath.Point{G}
	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			P[i][j] = make([]ecmath.Point, 1)
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))
			P[i][j][0].ScMul(&G, &p2)
			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n) // n zeroes
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}

// n=1 ring, m=1 sig, M=2 base points
func TestBorrRingSig112(t *testing.T) {
	msg := []byte("message")

	pbytes := [][]string{
		[]string{
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736",
		},
	}

	B := []ecmath.Point{G, J}

	n := len(pbytes)
	m := len(pbytes[0])
	M := len(B)

	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))

			P[i][j] = make([]ecmath.Point, M)

			for u := 0; u < M; u++ {
				P[i][j][u].ScMul(&B[u], &p2)
			}

			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n) // n zeroes
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}

// n=2 rings, m=1 sig, M=1 base point
func TestBorrRingSig211(t *testing.T) {
	msg := []byte("message")

	pbytes := [][]string{
		[]string{
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736",
		},
		[]string{
			"77bcedec0304c5239a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af9",
		},
	}

	B := []ecmath.Point{G}

	n := len(pbytes)
	m := len(pbytes[0])
	M := len(B)

	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))

			P[i][j] = make([]ecmath.Point, M)

			for u := 0; u < M; u++ {
				P[i][j][u].ScMul(&B[u], &p2)
			}

			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n) // n zeroes
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}

// n=3 rings, m=2 sig, M=1 base point
func TestBorrRingSig321(t *testing.T) {
	msg := []byte("message")

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

	B := []ecmath.Point{G}
	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		P[i] = make([][]ecmath.Point, m)
		for j := 0; j < m; j++ {
			P[i][j] = make([]ecmath.Point, 1)
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[i][j]))
			P[i][j][0].ScMul(&G, &p2)
			if j == 0 {
				p[i] = p2
			}
		}
	}
	j := make([]uint64, n) // n zeroes
	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, j, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}
