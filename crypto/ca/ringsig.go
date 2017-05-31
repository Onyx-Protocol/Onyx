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
// (Note: P has n elements; P[i] has M elements.)
func CreateRingSignature(msg []byte, M int, B []ecmath.Point, P [][]ecmath.Point, j uint64, p ecmath.Scalar) *RingSignature {
	return createRingSignature(msg, M, B, P, j, p, 0)
}

func createRingSignature(msg []byte, M int, B []ecmath.Point, P [][]ecmath.Point, j uint64, p ecmath.Scalar, counter uint64) *RingSignature {
	// 2. Let the msghash be a hash of the input non-secret data:
	// msghash = Hash256("RS" || byte(48+M) || B || P[0] || ... ||
	// P[n-1] || msg).
	msghash := rsMsgHash(M, B, P, msg)

	n := len(P)
	var (
		r     = make([]ecmath.Scalar, n-1)
		nonce [64]byte
		mask  [1]byte
	)
	stream := streamHash(uint64le(counter), msghash[:], p[:], uint64le(j))
	for i := 0; i < n-1; i++ {
		stream.Read(r[i][:])
	}
	stream.Read(nonce[:])
	stream.Read(mask[:])

	var k ecmath.Scalar
	k.Reduce(&nonce)

	R := make([][]ecmath.Point, n)
	i := uint64((j + 1) % uint64(n))
	R[i] = make([]ecmath.Point, M)
	for u := 0; u < M; u++ {
		R[i][u].ScMul(&B[u], &k)
	}
	w := mask[0] & 0xf0

	e := make([]ecmath.Scalar, n)
	e[i] = rsEHash(R[i], msghash[:], i, w)

	s := make([]ecmath.Scalar, n)
	for step := 1; step < n; step++ {
		i = uint64((j + uint64(step)) % uint64(n))
		s[i] = r[step-1]
		z := s[i]
		z[31] &= 0x0f
		w = s[i][31] & 0xf0
		iPrime := (i + 1) % uint64(n)
		R[iPrime] = make([]ecmath.Point, M)
		for u := 0; u < M; u++ {
			R[iPrime][u].ScMul(&B[u], &z)
			var R2 ecmath.Point
			R2.ScMul(&P[i][u], &e[i])
			R[iPrime][u].Sub(&R[iPrime][u], &R2)
		}
		e[iPrime] = rsEHash(R[iPrime], msghash[:], iPrime, w)
	}

	s[j].MulAdd(&p, &e[j], &k) // z = p * e[j] + k
	if s[j][31]&0xf0 != 0 {
		// s[j] > 2^252-1
		return createRingSignature(msg, M, B, P, j, p, counter+1)
	}
	s[j][31] &= 0x0f
	s[j][31] |= (mask[0] & 0xf0)

	return &RingSignature{
		e: e[0],
		s: s,
	}
}

func (rs *RingSignature) Validate(msg []byte, M int, B []ecmath.Point, P [][]ecmath.Point) bool {
	msghash := rsMsgHash(M, B, P, msg)
	n := len(P)
	e := make([]ecmath.Scalar, n+1)
	e[0] = rs.e
	for i := 0; i < n; i++ {
		z := rs.s[i]
		z[31] &= 0x0f
		w := rs.s[i][31] & 0xf0
		R := make([]ecmath.Point, M)
		for u := 0; u < M; u++ {
			R[u].ScMul(&B[u], &z)
			var R2 ecmath.Point
			R2.ScMul(&P[i][u], &e[i])
			R[u].Sub(&R[u], &R2)
		}
		e[i+1] = rsEHash(R, msghash[:], uint64(i+1), w)
	}
	return e[0] == e[n]
}

func rsMsgHash(M int, B []ecmath.Point, P [][]ecmath.Point, msg []byte) [32]byte {
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
