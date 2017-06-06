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
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83706",
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
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83706",
			"7b0e047166c159e10f2f0e4a16c159e10a1f379f2d6d83736a1f37f2f0e4a102",
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
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83706",
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
			"77bcedec0304c5239a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6a09",
		},
		[]string{
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83706",
		},
	}
	js := []uint64{
		0,
		0,
	}

	B := []ecmath.Point{G}

	n := len(pbytes)
	m := len(pbytes[0])
	M := len(B)

	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for t := 0; t < n; t++ {
		P[t] = make([][]ecmath.Point, m)
		for i := 0; i < m; i++ {
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[t][i]))

			P[t][i] = make([]ecmath.Point, M)

			for u := 0; u < M; u++ {
				P[t][i][u].ScMul(&B[u], &p2)
			}

			if i == int(js[t]) {
				p[t] = p2
			}
		}
	}

	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, js, payload)
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
			"59e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9dafa11a337b0e04716601",
		},
		[]string{
			"9a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6af977bcedec0304c503",
			"1a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83736b5ea5caeec9daf01",
		},
		[]string{
			"77bcedec0304c5239a87c7c71419da044f6d87e9e83dc1dc6868c13fa1ab6a09",
			"b5ea5caeec9dafa11a337b0e047166c159e10a1f37f2f0e4a1f289f2d6d83706",
		},
	}

	js := []uint64{
		0,
		0,
		0,
	}

	B := []ecmath.Point{G}

	n := len(pbytes)
	m := len(pbytes[0])
	M := len(B)

	p := make([]ecmath.Scalar, n)
	P := make([][][]ecmath.Point, n)
	for t := 0; t < n; t++ {
		P[t] = make([][]ecmath.Point, m)
		for i := 0; i < m; i++ {
			var p2 ecmath.Scalar
			hex.Decode(p2[:], []byte(pbytes[t][i]))

			P[t][i] = make([]ecmath.Point, M)

			for u := 0; u < M; u++ {
				P[t][i][u].ScMul(&B[u], &p2)
			}

			if i == int(js[t]) {
				p[t] = p2
			}
		}
	}

	payload := make([][32]byte, m*n)
	brs := CreateBorromeanRingSignature(msg, B, P, p, js, payload)
	if !brs.Validate(msg, B, P) {
		t.Error("failed to validate borromean ring signature")
	}
}

func TestBorrRingSig_n_m_M(t *testing.T) {
	msg := []byte("message")

	// Iterate different number of rings, signatures and base points
	for n := 1; n <= 8; n *= 2 { // choosing different number of rings
		for m := 1; m <= 4; m *= 2 { // choosing different number of signatures
			for M := 1; M <= 2; M++ { // choosing different number of base points

				// Generate n*m privkeys
				privkeys := make([][]ecmath.Scalar, n)
				for t := 0; t < n; t++ {
					privkeys[t] = make([]ecmath.Scalar, m)
					for i := 0; i < m; i++ {
						// Generates unique privkeys for each parameter combo
						privkeys[t][i] = scalarHash([]byte{byte(n), byte(m), byte(M), byte(t), byte(i)}, msg)
					}
				}

				// Generate basepoints
				basepoints := []ecmath.Point{G, J}
				basepoints = basepoints[0:M]

				// Iterate different secret indices (each of n rings has its own secret index j from 0 to m-1)
				// 0 => all zeroes
				// 1 => all m-1
				// 2 => 0,1,2,...
				// 3 => m-1,m-2,...2,1,0,m-1,...
				for c := 0; c < 4; c++ {
					js := make([]uint64, n)
					for t := 0; t < n; t++ {
						js[t] = 0
						if c == 1 {
							js[t] = uint64(m - 1)
						} else if c == 2 {
							js[t] = uint64(t % m) // 0 for ring 0, 1 for ring 1 etc.
						} else if c == 3 {
							js[t] = uint64(m - 1 - (t % m)) // m-1 for ring 0, m-2 for ring 1 etc.
						}
					}

					// Prepare privkeys and pubkeys for BRS

					p := make([]ecmath.Scalar, n)
					P := make([][][]ecmath.Point, n)
					for t := 0; t < n; t++ {
						P[t] = make([][]ecmath.Point, m)
						for i := 0; i < m; i++ {
							P[t][i] = make([]ecmath.Point, M)

							privkey := privkeys[t][i]

							for u := 0; u < M; u++ {
								P[t][i][u].ScMul(&basepoints[u], &privkey)
							}
							if i == int(js[t]) {
								p[t] = privkey
							}
						}
					}

					payload := make([][32]byte, m*n)
					//fmt.Printf("TEST: CreateBorromeanRingSignature (n=%d,m=%d,M=%d,c=%d)\n", n, m, M, c)
					brs := CreateBorromeanRingSignature(msg, basepoints, P, p, js, payload)
					if !brs.Validate(msg, basepoints, P) {
						t.Errorf("failed to validate borromean ring signature (n=%d,m=%d,M=%d,c=%d)", n, m, M, c)
					}
				}
			}
		}
	}
}
