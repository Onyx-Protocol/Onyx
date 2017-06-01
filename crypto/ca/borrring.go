package ca

import "chain/crypto/ed25519/ecmath"

type BorromeanRingSignature struct {
	e ecmath.Scalar
	s [][]ecmath.Scalar
}

// 1. msg: the string to be signed.
// 2. n: number of rings.
// 3. m: number of signatures in each ring.
// 4. M: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
// 5. {B[u]}: M base [points](#point) to validate the signature.
// 6. {P[i,j,u]}: n·m·M [points](#point) representing public keys.
// 7. {p[i]}: the list of n [scalars](#scalar) representing private keys.
// 8. {j[i]}: the list of n indexes of the designated public keys within each ring, so that P[i,j] == p[i]·B[i].
// 9. {payload[i]}: sequence of n·m random 32-byte elements.
func CreateBorromeanRingSignature(msg []byte, B []ecmath.Point, P [][][]ecmath.Point, p []ecmath.Scalar, j []uint64, payload [][32]byte) *BorromeanRingSignature {
	msghash := brsMsgHash(B, P, msg)
	return createBorromeanRingSignature(msghash[:], B, P, p, j, payload, 0)
}

func createBorromeanRingSignature(msghash []byte, B []ecmath.Point, P [][][]ecmath.Point, p []ecmath.Scalar, j []uint64, payload [][32]byte, counter uint64) *BorromeanRingSignature {
	n := uint64(len(P))
	m := uint64(len(P[0]))
	M := len(B)

	cnt := byte(counter & 0x0f)

	var (
		o = make([][32]byte, m*n)
		r = make([]ecmath.Scalar, m*n)
		w = make([][]byte, n)
	)
	stream := streamHash([]byte("O"), uint64le(counter), msghash)
	for _, pi := range p {
		stream.Write(pi[:])
	}
	for _, ji := range j {
		stream.Write(uint64le(ji))
	}
	for i := uint64(0); i < m*n; i++ {
		stream.Read(o[i][:])
		for j := 0; j < 32; j++ {
			r[i][j] = payload[i][j] ^ o[i][j]
		}
	}
	var (
		k    = make([]ecmath.Scalar, n)
		mask = make([]byte, n)
		e    = make([][]ecmath.Scalar, n)
		s    = make([][]ecmath.Scalar, n)
		z    = make([][]ecmath.Scalar, n)
	)
	for t := uint64(0); t < n; t++ {
		jt := j[t]
		x := r[m*t+jt]
		k[t] = x
		k[t][31] &= 0x0f
		mask[t] = x[31] & 0xf0
		w[t] = make([]byte, m) // xxx is "m" right here?
		w[t][jt] = mask[t]
		jPrime := (jt + 1) % m
		R := make([]ecmath.Point, M)
		for u := 0; u < M; u++ {
			R[u].ScMul(&B[u], &k[t])
		}
		e[t] = make([]ecmath.Scalar, m)
		e[t][jPrime] = brsEHash(cnt, R, msghash, t, jPrime, w[t][jt])

		s[t] = make([]ecmath.Scalar, m)
		z[t] = make([]ecmath.Scalar, m)
		for i := jt + 1; i < m; i++ {
			s[t][i] = r[m*t+i]

			z[t][i] = s[t][i]
			z[t][i][31] &= 0x0f

			w[t][i] = s[t][i][31] & 0xf0

			iPrime := (i + 1) % m
			e[t][iPrime] = brsNextE(B, P[t][i], z[t][i], e[t][i], msghash, t, iPrime, cnt, w[t][i])
		}
	}
	hasher := scalarHasher()
	for t := uint64(0); t < n; t++ {
		hasher.Write(e[t][0][:])
	}
	e0 := scalarHasherFinalize(hasher)
	if e0[31]&0xf0 != 0 {
		return createBorromeanRingSignature(msghash, B, P, p, j, payload, counter+1)
	}
	for t := uint64(0); t < n; t++ {
		jt := j[t]
		e[t][0] = e0
		for i := uint64(0); i < jt; i++ {
			s[t][i] = r[m*t+i]
			z[t][i] = s[t][i]
			z[t][i][31] &= 0x0f
			w[t][i] = s[t][i][31] & 0xf0
			iPrime := (i + 1) % m
			e[t][iPrime] = brsNextE(B, P[t][i], z[t][i], e[t][i], msghash, t, iPrime, cnt, w[t][i])
		}
		z[t][jt].MulAdd(&p[t], &e[t][jt], &k[t])
		if z[t][jt][31]&0xf0 != 0 {
			return createBorromeanRingSignature(msghash, B, P, p, j, payload, counter+1)
		}
		s[t][jt] = z[t][jt]
		s[t][jt][31] &= 0x0f
		s[t][jt][31] |= mask[t]
	}
	return &BorromeanRingSignature{e: e0, s: s}
}

func brsMsgHash(B []ecmath.Point, P [][][]ecmath.Point, msg []byte) [32]byte {
	n := uint64(len(P))
	m := uint64(len(P[0]))
	M := len(B)
	hasher := hasher256([]byte("BRS"), []byte{byte(48 + M)}, uint64le(n), uint64le(m))
	for _, Bi := range B {
		hasher.Write(Bi.Bytes())
	}
	for _, Pi := range P {
		for _, Pij := range Pi {
			for _, Piju := range Pij {
				hasher.Write(Piju.Bytes())
			}
		}
	}
	hasher.Write(msg)
	var msghash [32]byte
	hasher.Read(msghash[:])
	return msghash
}

func brsEHash(cnt byte, R []ecmath.Point, msghash []byte, t, i uint64, w byte) ecmath.Scalar {
	M := len(R)
	hasher := scalarHasher([]byte("e"), []byte{cnt})
	for u := 0; u < M; u++ {
		hasher.Write(R[u].Bytes())
	}
	hasher.Write(msghash)
	hasher.Write(uint64le(t))
	hasher.Write(uint64le(i))
	hasher.Write([]byte{w})
	return scalarHasherFinalize(hasher)
}

func brsNextE(B, P []ecmath.Point, z, e ecmath.Scalar, msghash []byte, t, i uint64, cnt, w byte) ecmath.Scalar {
	M := len(B)
	R := make([]ecmath.Point, M)
	for u := 0; u < M; u++ {
		R[u].ScMul(&B[u], &z)
		var R2 ecmath.Point
		R2.ScMul(&P[u], &e)
		R[u].Sub(&R[u], &R2)
	}
	return brsEHash(cnt, R, msghash, t, i, w)
}

func (brs *BorromeanRingSignature) Validate(msg []byte, B []ecmath.Point, P [][][]ecmath.Point) bool {
	msghash := brsMsgHash(B, P, msg)
	n := uint64(len(P))
	m := uint64(len(P[0]))

	hasher := scalarHasher()

	e0 := brs.e
	cnt := brs.e[31] >> 4
	e0[31] &= 0x0f
	var (
		e = make([][]ecmath.Scalar, n)
		z = make([][]ecmath.Scalar, n)
		w = make([][]byte, n)
	)
	for t := uint64(0); t < n; t++ {
		e[t] = make([]ecmath.Scalar, m)
		e[t][0] = e0
		z[t] = make([]ecmath.Scalar, m)
		w[t] = make([]byte, m)
		for i := uint64(0); i < m; i++ {
			z[t][i] = brs.s[t][i]
			z[t][i][31] &= 0x0f
			w[t][i] = brs.s[t][i][31] & 0xf0
			iPrime := (i + 1) % m
			e[t][iPrime] = brsNextE(B, P[t][i], z[t][i], e[t][i], msghash[:], t, iPrime, cnt, w[t][i])
		}
		hasher.Write(e[t][0][:])
	}
	ePrime := scalarHasherFinalize(hasher)
	return ePrime == e0
}
