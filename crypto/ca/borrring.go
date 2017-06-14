package ca

import (
	"fmt"

	"chain/crypto/ed25519/ecmath"
)

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
	return createBorromeanRingSignature(msg, B, P, p, j, payload, 0)
}

func createBorromeanRingSignature(msg []byte, B []ecmath.Point, P [][][]ecmath.Point, p []ecmath.Scalar, j []uint64, payload [][32]byte, counter uint64) *BorromeanRingSignature {
	msghash := brsMsgHash(B, P, msg)

	n := uint64(len(P))
	if n < 1 {
		panic("number of rings cannot be less than 1")
	}

	m := uint64(len(P[0]))
	M := uint64(len(B))

	if m < 1 {
		panic("number of signatures per ring cannot be less than 1")
	}
	if uint64(len(p)) != n {
		panic("number of secret keys must equal number of rings")
	}
	if uint64(len(j)) != n {
		panic("number of secret indexes must equal number of rings")
	}
	if uint64(len(payload)) != n*m {
		panic("number of payload elements must equal n*m (rings*signatures)")
	}

	cnt := byte(counter & 0x0f)

	o := brsOverlay(counter, msghash[:], p, j, m)

	r := make([]ecmath.Scalar, m*n)
	for i := uint64(0); i < m*n; i++ {
		for j := 0; j < 32; j++ {
			r[i][j] = payload[i][j] ^ o[i][j]
		}
	}

	var (
		k    = make([]ecmath.Scalar, n)
		mask = make([]byte, n)
		s    = make([][]ecmath.Scalar, n)
		e    ecmath.Scalar
		w    byte
	)
	e0hasher := scalarHasher("ChainCA.BRS.e0")
	for t := uint64(0); t < n; t++ {
		jt := j[t]
		x := r[m*t+jt]
		k[t] = x
		k[t][31] &= 0x0f
		mask[t] = x[31] & 0xf0

		// 6.5. Define `w[t,j]` as a byte with lower 4 bits set to zero and higher 4 bits equal `mask[t]`.
		w = mask[t]

		jPrime := (jt + 1) % m
		R := make([]ecmath.Point, M)
		for u := uint64(0); u < M; u++ {
			R[u].ScMul(&B[u], &k[t])
		}
		e = brsEHash(cnt, R, msghash[:], t, jPrime, w)

		s[t] = make([]ecmath.Scalar, m)
		for i := jt + 1; i < m; i++ {
			s[t][i] = r[m*t+i]

			z := s[t][i]
			z[31] &= 0x0f

			w = s[t][i][31] & 0xf0

			iPrime := (i + 1) % m
			e = brsNextE(B, P[t][i], z, e, msghash[:], t, iPrime, cnt, w)
		}
		e0hasher.Write(e[:])
	}
	e0 := scalarHasherFinalize(e0hasher)
	if e0[31]&0xf0 != 0 {
		return createBorromeanRingSignature(msg, B, P, p, j, payload, counter+1)
	}
	for t := uint64(0); t < n; t++ {
		jt := j[t]
		e = e0
		for i := uint64(0); i < jt; i++ {
			s[t][i] = r[m*t+i]
			z := s[t][i]
			z[31] &= 0x0f
			w = s[t][i][31] & 0xf0
			iPrime := (i + 1) % m
			e = brsNextE(B, P[t][i], z, e, msghash[:], t, iPrime, cnt, w)
		}
		var z ecmath.Scalar
		z.MulAdd(&p[t], &e, &k[t])
		if z[31]&0xf0 != 0 {
			return createBorromeanRingSignature(msg, B, P, p, j, payload, counter+1)
		}
		s[t][jt] = z
		s[t][jt][31] &= 0x0f
		s[t][jt][31] |= mask[t]
	}
	// 9. Set top 4 bits of `e0` to the lower 4 bits of `counter`.
	counterByte := byte(counter & 0xff)
	e0[31] |= ((counterByte << 4) & 0xf0)

	return &BorromeanRingSignature{e: e0, s: s}
}

func (brs *BorromeanRingSignature) Validate(msg []byte, B []ecmath.Point, P [][][]ecmath.Point) bool {
	msghash := brsMsgHash(B, P, msg)

	n := uint64(len(P))
	if n < 1 {
		panic("number of rings cannot be less than 1")
	}

	m := uint64(len(P[0]))
	M := uint64(len(B))

	if m < 1 {
		panic("number of signatures per ring cannot be less than 1")
	}

	e0 := brs.e
	cnt := e0[31] >> 4
	e0[31] &= 0x0f

	var (
		e ecmath.Scalar
		z ecmath.Scalar
		w byte
	)
	e0hasher := scalarHasher("ChainCA.BRS.e0")
	for t := uint64(0); t < n; t++ {
		if uint64(len(P[t])) != m {
			panic(fmt.Errorf("number of pubkeys per ring must be %d*%d", m, M))
		}
		e = e0
		for i := uint64(0); i < m; i++ {
			if uint64(len(P[t][i])) != M {
				panic(fmt.Errorf("number of pubkeys per signature must be %d", M))
			}

			z = brs.s[t][i]
			z[31] &= 0x0f
			w = brs.s[t][i][31] & 0xf0

			iPrime := (i + 1) % m
			e = brsNextE(B, P[t][i], z, e, msghash[:], t, iPrime, cnt, w)
		}
		e0hasher.Write(e[:])
	}
	ePrime := scalarHasherFinalize(e0hasher)
	return ePrime == e0
}

// Payload recovers the inline data stored within a rangeproof
func (brs *BorromeanRingSignature) Payload(msg []byte, B []ecmath.Point, P [][][]ecmath.Point, p []ecmath.Scalar, j []uint64) [][32]byte {
	msghash := brsMsgHash(B, P, msg)

	n := uint64(len(P))
	if n < 1 {
		panic("number of rings cannot be less than 1")
	}

	m := uint64(len(P[0]))
	M := uint64(len(B))

	if m < 1 {
		panic("number of signatures per ring cannot be less than 1")
	}
	if uint64(len(p)) != n {
		panic("number of secret keys must equal number of rings")
	}
	if uint64(len(j)) != n {
		panic("number of secret indexes must equal number of rings")
	}

	e0 := brs.e
	cnt := e0[31] >> 4
	e0[31] &= 0x0f

	o := brsOverlay(uint64(cnt), msghash[:], p, j, m)
	payload := make([][32]byte, n*m)

	var (
		e   ecmath.Scalar
		z   ecmath.Scalar
		k   ecmath.Scalar
		tmp ecmath.Scalar
		w   byte
	)
	e0hasher := scalarHasher("ChainCA.BRS.e0")
	for t := uint64(0); t < n; t++ {
		if uint64(len(P[t])) != m {
			panic(fmt.Errorf("number of pubkeys per ring must be %d*%d", m, M))
		}
		e = e0
		for i := uint64(0); i < m; i++ {
			if uint64(len(P[t][i])) != M {
				panic(fmt.Errorf("number of pubkeys per signature must be %d", M))
			}

			z = brs.s[t][i]
			z[31] &= 0x0f
			w = brs.s[t][i][31] & 0xf0

			// Recover payload from either secret nonce or the forged s-element
			// 3. If `i` is equal to `j[t]`:
			if i == j[t] {
				// 	1. Calculate `k[t] = z[t,i] - p[t]·e[t,i] mod L`.
				tmp.MulAdd(&p[t], &e, &ecmath.Zero) // tmp = p*e+0
				tmp.Neg(&tmp)                       // tmp = -tmp
				k.Add(&z, &tmp)                     // k   = z + tmp

				// 	2. Set top 4 bits of `k[t]` to the top 4 bits of `w[t,i]`: `k[t][31] |= w[t,i]`.
				k[31] |= w

				// 	3. Set `payload[m·t + i] = o[m·t + i] XOR k[t]`.
				for b := 0; b < 32; b++ {
					payload[m*t+i][b] = o[m*t+i][b] ^ k[b]
				}

				// 4. If `i` is not equal to `j[t]`:
			} else {
				// 	1. Set `payload[m·t + i] = o[m·t + i] XOR s[t,i]`.
				for b := 0; b < 32; b++ {
					payload[m*t+i][b] = o[m*t+i][b] ^ brs.s[t][i][b]
				}
			}

			// Continue validating the signature
			iPrime := (i + 1) % m
			e = brsNextE(B, P[t][i], z, e, msghash[:], t, iPrime, cnt, w)
		}
		e0hasher.Write(e[:])
	}
	ePrime := scalarHasherFinalize(e0hasher)
	if ePrime != e0 {
		return nil
	}
	return payload
}

func brsMsgHash(B []ecmath.Point, P [][][]ecmath.Point, msg []byte) [32]byte {
	n := uint64(len(P))
	m := uint64(len(P[0]))
	M := len(B)
	hasher := hasher256("ChainCA.BRS.msg", []byte{byte(48 + M)}, uint64le(n), uint64le(m))
	for _, Bi := range B {
		hasher.WriteItem(Bi.Bytes())
	}
	for _, Pi := range P {
		for _, Pij := range Pi {
			for _, Piju := range Pij {
				hasher.WriteItem(Piju.Bytes())
			}
		}
	}
	hasher.WriteItem(msg)
	var msghash [32]byte
	hasher.Sum(msghash[:0])
	return msghash
}

func brsEHash(cnt byte, R []ecmath.Point, msghash []byte, t, i uint64, w byte) ecmath.Scalar {
	M := len(R)
	hasher := scalarHasher("ChainCA.BRS.e", []byte{cnt})
	for u := 0; u < M; u++ {
		hasher.WriteItem(R[u].Bytes())
	}
	hasher.WriteItem(msghash)
	hasher.WriteItem(uint64le(t))
	hasher.WriteItem(uint64le(i))
	hasher.WriteItem([]byte{w})

	e := scalarHasherFinalize(hasher)
	return e
}

func brsNextE(B, P []ecmath.Point, z, e ecmath.Scalar, msghash []byte, t, i uint64, cnt, w byte) ecmath.Scalar {
	M := len(B)
	R := make([]ecmath.Point, M)
	for u := 0; u < M; u++ {
		// R = z*B - e*P
		R[u].ScMul(&B[u], &z)
		var R2 ecmath.Point
		R2.ScMul(&P[u], &e)
		R[u].Sub(&R[u], &R2)
	}
	return brsEHash(cnt, R, msghash, t, i, w)
}

func brsOverlay(counter uint64, msghash []byte, p []ecmath.Scalar, j []uint64, m uint64) [][32]byte {
	n := uint64(len(p))
	stream := streamHash("ChainCA.BRS.Overlay", uint64le(counter), msghash)
	for _, pi := range p {
		stream.WriteItem(pi[:])
	}
	for _, ji := range j {
		stream.WriteItem(uint64le(ji))
	}
	result := make([][32]byte, m*n)
	for i := uint64(0); i < m*n; i++ {
		stream.Read(result[i][:])
	}
	return result
}
