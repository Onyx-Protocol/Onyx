package ca

import "chain/crypto/ed25519/ecmath"

// The ring signature is encoded as a string of n+1 32-byte elements
// where n is the number of public keys provided separately (typically
// stored or imputed from the data structure containing the ring
// signature):
//   {e, s[0], s[1], ..., s[n-1]}
// Each 32-byte element is an integer coded using little endian
// convention. I.e., a 32-byte string x x[0],...,x[31] represents the
// integer x[0] + 2^8 · x[1] + ... + 2^248 · x[31].
type RingSignature struct {
	e ecmath.Scalar
	s []ecmath.Scalar
}

// msg: the string to be signed.
// M: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
// {B[u]}: M base points to validate the signature.
// {P[i,u]}: n·M points representing the public keys.
// j: the index of the designated public key, so that P[j] == p·B.
// p: the secret scalar representing a private key for the public keys P[u,j].
//
// (Layout note: P has n elements; P[i] has M elements.)
func CreateRingSignature(msg []byte, B []ecmath.Point, P [][]ecmath.Point, j uint64, p ecmath.Scalar) *RingSignature {
	// 1. Let counter = 0.
	// 2. Let the msghash be a hash of the input non-secret data:
	// msghash = Hash256("RS" || byte(48+M) || B || P[0] || ... ||
	// P[n-1] || msg).
	msghash := rsMsgHash(B, P, msg)
	return createRingSignature(msghash[:], B, P, j, p, 0)
}

func createRingSignature(msghash []byte, B []ecmath.Point, P [][]ecmath.Point, j uint64, p ecmath.Scalar, counter uint64) *RingSignature {
	n := uint64(len(P))
	M := len(B)

	// 3. Calculate a sequence of: n-1 32-byte random values, 64-byte
	// nonce and 1-byte mask: {r[i], nonce, mask} =
	// StreamHash(uint64le(counter) || msghash || p || uint64le(j),
	// 32·(n-1) + 64 + 1)
	var (
		r     = make([]ecmath.Scalar, n-1)
		nonce [64]byte
		mask  [1]byte
	)
	stream := streamHash(uint64le(counter), msghash[:], p[:], uint64le(j)) // xxx should this start with some prefix?
	for i := uint64(0); i < n-1; i++ {
		stream.Read(r[i][:])
	}
	stream.Read(nonce[:])
	stream.Read(mask[:])

	// 4. Calculate k = nonce mod L, where nonce is interpreted as a
	// 64-byte little-endian integer and reduced modulo subgroup order
	// L.
	var k ecmath.Scalar
	k.Reduce(&nonce)

	// 5. Calculate the initial e-value, let i = j+1 mod n
	R := make([]ecmath.Point, M) // impl. note: this R is the spec's R[i]
	i := (j + 1) % n
	for u := 0; u < M; u++ {
		// 5.1. For each u from 0 to M-1: calculate R[i,u] as the [point](#point) k·B[u].
		R[u].ScMul(&B[u], &k)
	}
	// 5.2. Define w[j] as mask with lower 4 bits set to zero: w[j] = mask & 0xf0.
	w := mask[0] & 0xf0
	// 5.3. Calculate e[i] = ScalarHash("e" || R[i,0] || ... || R[i,M-1] || msghash || uint64le(i) || w[j]).
	e := make([]ecmath.Scalar, n) // really we only need to save e[0] and the last e in each loop iteration below
	e[i] = rsEHash(R, msghash[:], i, w)

	s := make([]ecmath.Scalar, n)
	// 6. For step from 1 to n-1:
	for step := uint64(1); step < n; step++ {
		// 6.1. Let i = (j + step) mod n.
		i = (j + step) % n
		// 6.2. Calculate the forged s-value s[i] = r[step-1].
		s[i] = r[step-1]
		// 6.3. Define z[i] as s[i] with the most significant 4 bits set to zero.
		z := s[i]
		z[31] &= 0x0f
		// 6.4. Define w[i] as a most significant byte of s[i] with lower 4 bits set to zero: w[i] = s[i][31] & 0xf0.
		w = s[i][31] & 0xf0
		// 6.5. Let i’ = i+1 mod n.
		iPrime := (i + 1) % n
		e[iPrime] = rsNextE(msghash[:], iPrime, B, z, P[i], e[i], w)
	}

	s[j].MulAdd(&p, &e[j], &k) // z = p * e[j] + k
	if s[j][31]&0xf0 != 0 {
		// s[j] > 2^252-1
		return createRingSignature(msghash, B, P, j, p, counter+1)
	}
	s[j][31] &= 0x0f
	s[j][31] |= (mask[0] & 0xf0)

	return &RingSignature{
		e: e[0],
		s: s,
	}
}

func (rs *RingSignature) Validate(msg []byte, B []ecmath.Point, P [][]ecmath.Point) bool {
	msghash := rsMsgHash(B, P, msg)

	n := uint64(len(P))

	e := make([]ecmath.Scalar, n+1)
	e[0] = rs.e
	for i := uint64(0); i < n; i++ {
		z := rs.s[i]
		z[31] &= 0x0f
		w := rs.s[i][31] & 0xf0
		e[i+1] = rsNextE(msghash[:], i+1, B, z, P[i], e[i], w)
	}
	return e[0] == e[n]
}

func rsMsgHash(B []ecmath.Point, P [][]ecmath.Point, msg []byte) [32]byte {
	M := len(B)

	hasher := hasher256([]byte("RS"), []byte{byte(48 + M)})
	for _, b := range B {
		hasher.Write(b.Bytes())
	}
	for i := range P { // xxx check this is the right ordering (P[0][0], P[0][1], P[1][0], ...)
		for u := 0; u < M; u++ {
			hasher.Write(P[i][u].Bytes())
		}
	}
	hasher.Write(msg)
	var msghash [32]byte
	hasher.Read(msghash[:])
	return msghash
}

func rsEHash(R []ecmath.Point, msghash []byte, i uint64, w byte) ecmath.Scalar {
	scHasher := scalarHasher([]byte("e"))
	for u := 0; u < len(R); u++ {
		scHasher.Write(R[u].Bytes())
	}
	scHasher.Write(msghash)
	scHasher.Write(uint64le(i))
	scHasher.Write([]byte{w})
	return scalarHasherFinalize(scHasher)
}

// note: P is just one row of the caller's (two-dimensional) P
func rsNextE(msghash []byte, i uint64, B []ecmath.Point, z ecmath.Scalar, P []ecmath.Point, prevE ecmath.Scalar, w byte) ecmath.Scalar {
	M := len(B)
	R := make([]ecmath.Point, M)
	for u := 0; u < M; u++ {
		R[u].ScMul(&B[u], &z)
		var R2 ecmath.Point
		R2.ScMul(&P[u], &prevE)
		R[u].Sub(&R[u], &R2)
	}
	return rsEHash(R, msghash, i, w)
}
