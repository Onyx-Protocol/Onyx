package ca

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// Single ring signature
// The ring signature is encoded as a string of `n+1` 32-byte elements where `n` is the number of public keys
//     {e, s[0], s[1], ..., s[n-1]}
//
// Each 32-byte element is an integer coded using little endian convention.
// I.e., a 32-byte string `x` `x[0],...,x[31]` represents the integer `x[0] + 2^8 * x[1] + ... + 2^248 * x[31]`.
type ringSignature struct {
	e [32]byte
	s [][32]byte
}

func (rs *ringSignature) nPubkeys() uint32 {
	return uint32(len(rs.s))
}

func (r *ringSignature) writeTo(w io.Writer) error {
	_, err := w.Write(r.e[:])
	if err != nil {
		return err
	}
	for _, s := range r.s {
		_, err = w.Write(s[:])
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *ringSignature) readFrom(r io.Reader, n uint32) (err error) {
	_, err = io.ReadFull(r, rs.e[:])
	if err != nil {
		return err
	}
	rs.s = make([][32]byte, n)
	for i := uint32(0); i < n; i++ {
		_, err = io.ReadFull(r, rs.s[i][:])
		if err != nil {
			return err
		}
	}
	return nil
}

// This is an internal function that avoid unnecessary serialization for pubkeys.
// Inputs:
//
// 1. `msg`: the 32-byte string to be signed.
// 2. `{P[i]}`: `n` public keys, [points](data.md#public-key) on the elliptic curve.
// 3. `j`: the index of the designated public key, so that `P[j] == p*G`.
// 4. `p`: the private key for the public key `P[j]`.
//
// Output: `{e0, s[0], ..., s[n-1]}`: the ring signature, `n+1` 32-byte elements.
func createRingSignature(
	msg [32]byte,
	P []Point,
	j int,
	p Scalar,
) *ringSignature {
	if len(P) == 0 {
		ringsig := new(ringSignature)
		ringsig.s = make([][32]byte, 0)
		return ringsig
	}

	n := uint64(len(P))
	ringsig := new(ringSignature)
	ringsig.s = make([][32]byte, n)

	// 1. Let `counter = 0`.
	counter := uint64(0)
	for {
		e0 := [2]Scalar{} // second slot is to put non-zero value in a time-constant manner

		// 2. Calculate a sequence of: `n-1` 32-byte random values, 64-byte `nonce` and 1-byte `mask`:
		//    `{r[i], nonce, mask} = SHAKE256(counter || p || msg, 8*(32*(n-1) + 64 + 1))`,
		//    where `p` is encoded in 32 bytes using little-endian convention, and `counter` is encoded as a 64-bit little-endian integer.
		r := make([][32]byte, n)

		rhash := shake256(uint64le(counter), msg[:], p[:], uint64le(uint64(j)))
		for _, pubkey := range P {
			penc := encodePoint(&pubkey)
			rhash.Write(penc[:])
		}
		for i := uint64(0); i < (n - 1); i++ {
			rhash.Read(r[i][:])
		}
		var nonce [64]byte
		var mask [1]byte
		rhash.Read(nonce[:])
		rhash.Read(mask[:])

		// 3. Calculate `k = nonce mod L`, where `nonce` is interpreted as a 64-byte little-endian integer and reduced modulo subgroup order `L`.
		k := reducedScalar(nonce)

		// 4. Calculate the initial e-value, let `i = j+1 mod n`:
		i := (uint64(j) + 1) % n

		// 4.1. Calculate `R[i]` as the point `k*G`.
		Ri := multiplyBasePoint(k)
		// 4.2. Define `w[j]` as `mask` with lower 4 bits set to zero: `w[j] = mask & 0xf0`.
		wj := mask[0] & 0xf0
		// 4.3. Calculate `e[i] = SHA3-512(R[i] || msg || i)` where `i` is encoded as a 64-bit little-endian integer. Interpret `e[i]` as a little-endian integer reduced modulo `L`.
		ei := computeE(&Ri, msg[:], i, wj)
		if i == 0 {
			e0[0] = ei
		} else {
			e0[1] = ei
		}

		// 5. For `step` from `1` to `n-1` (these steps are skipped if `n` equals 1):
		for step := uint64(1); step < n; step++ {
			// 5.1. Let `i = (j + step) mod n`.
			i := (uint64(j) + step) % n

			// 5.2. Set the forged s-value `s[i] = r[step-1]`
			copy(ringsig.s[i][:], r[step-1][:])

			// 5.3. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero.
			z := ringsig.s[i]
			z[31] &= 0x0f

			// 5.4. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
			wi := ringsig.s[i][31] & 0xf0

			// 5.5. Let `i’ = i+1 mod n`.
			i1 := (i + 1) % n

			// 5.6. Calculate `R[i’] = z[i]*G - e[i]*P[i]` and encode it as a 32-byte public key.
			Ri1 := multiplyAndAddPoint(negateScalar(ei), P[i], z)

			// 5.7. Calculate `e[i’] = SHA3-512(R[i’] || msg || i’)` where `i’` is encoded as a 64-bit little-endian integer.
			// Interpret `e[i’]` as a little-endian integer.
			ei = computeE(&Ri1, msg[:], i1, wi)

			if i1 == 0 {
				e0[0] = ei
			} else {
				e0[1] = ei
			}
		}

		// 6. Calculate the non-forged `z[j] = k + p*e[j] mod L` and encode it as a 32-byte little-endian integer.
		zj := multiplyAndAddScalars(p, ei, k)

		// 7. If `z[j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from the beginning.
		//    The chance of this happening is below 1 in 2<sup>124</sup>.
		if (zj[31] & 0xf0) != 0 {

			// We won a lottery and will try again with an incremented counter.
			counter++

		} else {
			// 8. Define `s[j]` as `z[j]` with 4 high bits set to high 4 bits of the `mask`.
			zj[31] ^= (mask[0] & 0xf0) // zj now == sj

			// Put non-forged s[j] into ringsig
			copy(ringsig.s[j][:], zj[:])

			// Put e[0] inside the ringsig
			copy(ringsig.e[:], e0[0][:])

			break
		}
	}
	// 9. Return the ring signature `{e[0], s[0], ..., s[n-1]}`, total `n+1` 32-byte elements.
	return ringsig
}

// Verify Ring Signature
//
// Inputs:
//
// 1. `msg`: the 32-byte string being signed.
// 2. `e[0], s[0], ... s[n-1]`: ring signature consisting of `n+1` 32-byte little-endian integers.
// 3. `{P[i]}`: `n` public keys, [points](data.md#public-key) on the elliptic curve.
//
// Output: `true` if the verification succeeded, `false` otherwise.
func (rs *ringSignature) verify(
	msg [32]byte,
	P []Point,
) error {
	// 0. Sanity checks
	if len(rs.s) != len(P) {
		return fmt.Errorf("ring size %d does not equal number of pubkeys %d", len(rs.s), len(P))
	}

	// 1. For each `i` from `0` to `n-1`:
	n := uint64(len(P))
	e := rs.e

	for i := uint64(0); i < n; i++ {

		// 1. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero (see note below).
		var z [32]byte
		copy(z[:], rs.s[i][:])
		z[31] &= 0x0f

		// 2. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
		w := rs.s[i][31] & 0xf0

		// 3. Calculate `R[i+1] = z[i]*G - e[i]*P[i]` and encode it as a 32-byte public key.
		R := multiplyAndAddPoint(negateScalar(e), P[i], z)

		// 4. Calculate `e[i+1] = SHA3-512(R[i+1] || msg || i+1)` where `i+1` is encoded as a 64-bit little-endian integer.
		// 5. Interpret `e[i+1]` as a little-endian integer reduced modulo subgroup order `L`.
		e = computeE(&R, msg[:], (i+1)%n, w)
	}

	// 4. Return true if `e[0]` equals `e[n]`, otherwise return false.
	if !constTimeEqual(e[:], rs.e[:]) {
		return fmt.Errorf("ringsig unverified")
	}
	return nil
}

// Calculates `e[i] = SHA3-512(R[i] || msg || i)` where `i` is encoded as a 64-bit little-endian integer.
// Interpret `e[i]` as a little-endian integer reduced modulo `L`.
func computeE(R *Point, msg []byte, i uint64, w byte) Scalar {
	Renc := encodePoint(R)
	ienc := uint64le(i)
	return reducedScalar(hash512(Renc[:], msg, ienc[:], []byte{w}))
}

func (rs ringSignature) String() string {
	sstrs := make([]string, 0, len(rs.s))
	for _, s := range rs.s {
		sstrs = append(sstrs, hex.EncodeToString(s[:]))
	}
	return fmt.Sprintf("{e: %x, s: [%s]}", rs.e[:], strings.Join(sstrs, ", "))
}
