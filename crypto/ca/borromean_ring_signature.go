package ca

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// Borromean ring signature
// The borromean ring signature is encoded as a sequence of n*m 32-byte elements where
// `n` is a number of rings and `m` is a number of public keys per ring.
//     {e, s[i,j]...}
// Where:
// * `i` is in range `0..n-1`
// * `j` is in range `0..m-1`
//
// Each 32-byte element is an integer coded using little endian convention.
// I.e., a 32-byte string `x` `x[0],...,x[31]` represents the integer `x[0] + 2^8 * x[1] + ... + 2^248 * x[31]`.
type borromeanRingSignature struct {
	e [32]byte
	s [][][32]byte
}

func (brs *borromeanRingSignature) writeTo(w io.Writer) error {
	_, err := w.Write(brs.e[:])
	if err != nil {
		return err
	}
	for _, outer := range brs.s {
		for _, s := range outer {
			_, err = w.Write(s[:])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (brs *borromeanRingSignature) readFrom(r io.Reader, nRings, nPubkeys int) error {
	_, err := io.ReadFull(r, brs.e[:])
	if err != nil {
		return err
	}
	brs.s = make([][][32]byte, nRings)
	for i := 0; i < nRings; i++ {
		brs.s[i] = make([][32]byte, nPubkeys)
		for j := 0; j < nPubkeys; j++ {
			_, err = io.ReadFull(r, brs.s[i][j][:])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Inputs:
// 1. `msg`: the 32-byte string to be signed.
// 2. `n`: number of rings.
// 3. `m`: number of signatures in each ring.
// 4. `{P[i,j]}`: `n*m` public keys, [points](data.md#public-key) on the elliptic curve.
// 5. `{p[i]}`: the list of `n` private keys.
// 6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]*G`.
// 7. `{r[v]}`: random string consisting of `n*m` 32-byte elements.
func createBorromeanRingSignature(
	msg [32]byte,
	pubkeys [][]Point,
	privkeys []Scalar,
	indexes []int,
	payload [][32]byte,
) (brs *borromeanRingSignature, err error) {
	n := len(pubkeys)
	if n < 1 {
		return brs, fmt.Errorf("number of rings cannot be less than 1")
	}
	m := len(pubkeys[0])
	if m < 1 {
		return brs, fmt.Errorf("number of signatures per ring cannot be less than 1")
	}
	if len(privkeys) != n {
		return brs, fmt.Errorf("number of secret keys must equal number of rings")
	}
	if len(indexes) != n {
		return brs, fmt.Errorf("number of secret indexes must equal number of rings")
	}
	if len(payload) != n*m {
		return brs, fmt.Errorf("number of random elements must equal n*m (rings*signatures)")
	}

	// 1. Let `counter = 0`.
	counter := uint64(0)
	for {
		var w byte
		s := make([][][32]byte, n)
		k := make([][32]byte, n)
		mask := make([]byte, n)
		E := hasher512()

		// 2. Let `cnt` byte contain lower 4 bits of `counter`: `cnt = counter & 0x0f`.
		cnt := byte(counter & 0x0f)

		// 3. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = SHAKE256(counter || msg || {p[i]} || {j[i]} || {P[i,j]}, 8·(32·n·m))`, where:
		//     * `counter` is encoded as a 64-bit little-endian integer,
		//     * private keys `{p[i]}` are encoded as concatenation of 256-bit little-endian integers,
		//     * secret indexes `{j[i]}` are encoded as concatenation of 64-bit little-endian integers,
		//     * points `{P[i]}` are encoded as concatenation of [public keys](data.md#public-key).
		overlay := shake256(uint64le(counter), msg[:])
		for i := 0; i < n; i++ {
			overlay.Write(privkeys[i][:])
		}
		for i := 0; i < n; i++ {
			overlay.Write(uint64le(uint64(indexes[i])))
		}
		for i := 0; i < n; i++ {
			for j := 0; j < m; j++ {
				pBytes := encodePoint(&pubkeys[i][j])
				overlay.Write(pBytes[:])
			}
		}

		// 4. Define `r[i] = payload[i] XOR o[i]` for all `i` from 0 to `n·m - 1`.
		r := make([][32]byte, n*m)
		for i := 0; i < n*m; i++ {
			overlay.Read(r[i][:])
			r[i] = xor256(r[i][:], payload[i][:])
		}

		// 5. For `t` from `0` to `n-1` (each ring):
		for t := 0; t < n; t++ {
			s[t] = make([][32]byte, m)

			// 5.1. Let `j = j[t]`
			j := indexes[t]

			// 5.2. Let `x = r[m·t + j]` interpreted as a little-endian integer.
			x := r[m*t+j]

			// 5.3. Define `k[t]` as the lower 252 bits of `x`.
			k[t] = x
			k[t][31] &= 0x0f

			// 5.4. Define `mask[t]` as the higher 4 bits of `x`.
			mask[t] = x[31] & 0xf0

			// 5.5. Define `w[t,j]` as a byte with lower 4 bits set to zero and higher 4 bits equal `mask[t]`.
			w = mask[t]

			// 5.6. Calculate the initial e-value for the ring:

			// 5.6.1. Let `j’ = j+1 mod m`.
			j1 := (j + 1) % m

			// 5.6.2. Calculate `R[t,j’]` as the point `k[t]*G` and encode it as a 32-byte [public key](data.md#public-key).
			R := multiplyBasePoint(k[t])

			// 5.6.3. Calculate `e[t,j’] = SHA3-512(R[t, j’] || msg || t || j’ || w[t,j])` where `t` and `j’` are encoded as 64-bit little-endian integers. Interpret `e[t,j’]` as a little-endian integer reduced modulo `L`.
			e := computeInnerE(cnt, &R, msg[:], uint64(t), uint64(j1), w)

			// 5.7. If `j ≠ m-1`, then for `i` from `j+1` to `m-1`:
			for i := j + 1; i < m; i++ { // note that j+1 can be == m in which case loop is empty as we need it to be.
				// 5.7.1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
				s[t][i] = r[m*t+i]

				// 5.7.2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
				z := s[t][i]
				z[31] &= 0xf

				// 5.7.3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
				w = s[t][i][31] & 0xf0

				// 5.7.4. Let `i’ = i+1 mod m`.
				i1 := (i + 1) % m

				// 5.7.5. Calculate point `R[t,i’] = z[t,i]*G - e[t,i]*P[t,i]` and encode it as a 32-byte [public key](data.md#public-key).
				R = multiplyAndAddPoint(negateScalar(e), pubkeys[t][i], z)

				// 5.7.6. Calculate `e[t,i’] = SHA3-512(R[t,i’] || msg || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers. Interpret `e[t,i’]` as a little-endian integer reduced modulo `L`.
				e = computeInnerE(cnt, &R, msg[:], uint64(t), uint64(i1), w)
			}

			// 6. Calculate the shared e-value `e0` for all the rings:
			// 6.1. Calculate `E` as concatenation of all `e[t,0]` values encoded as 32-byte little-endian integers: `E = e[0,0] || ... || e[n-1,0]`.
			E.Write(e[:])
		}

		// 6.2. Calculate `e0 = SHA3-512(E)`. Interpret `e0` as a little-endian integer reduced modulo `L`.
		var e0hash [64]byte
		E.Sum(e0hash[:0])
		e0 := reducedScalar(e0hash)

		// 6.3. If `e0` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from step 2.
		//      The chance of this happening is below 1 in 2<sup>124</sup>.
		if e0[31]&0xf0 != 0 {
			counter++
			continue
		}

		// 7. For `t` from `0` to `n-1` (each ring):
		for t := 0; t < n; t++ {
			// 7.1. Let `j = j[t]`
			j := indexes[t]

			// 7.2. Let `e[t,0] = e0`.
			e := e0

			// 7.3. If `j` is not zero, then for `i` from `0` to `j-1`:
			for i := 0; i < j; i++ {
				// 7.3.1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
				s[t][i] = r[m*t+i]

				// 7.3.2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
				z := s[t][i]
				z[31] &= 0x0f

				// 7.3.3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
				w = s[t][i][31] & 0xf0

				// 7.3.4. Let `i’ = i+1 mod m`.
				i1 := (i + 1) % m

				// 7.3.5. Calculate point `R[t,i’] = z[t,i]*G - e[t,i]*P[t,i]` and encode it as a 32-byte [public key](data.md#public-key). If `i` is zero, use `e0` in place of `e[t,0]`.
				R := multiplyAndAddPoint(negateScalar(e), pubkeys[t][i], z)

				// 7.3.6. Calculate `e[t,i’] = SHA3-512(R[t,i’] || msg || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers. Interpret `e[t,i’]` as a little-endian integer reduced modulo subgroup order `L`.
				e = computeInnerE(cnt, &R, msg[:], uint64(t), uint64(i1), w)
			}

			// 7.4. Calculate the non-forged `z[t,j] = k[t] + p[t]*e[t,j] mod L` and encode it as a 32-byte little-endian integer.
			z := multiplyAndAddScalars(privkeys[t], e, k[t])

			// 7.5. If `z[t,j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from step 2.
			//      The chance of this happening is below 1 in 2<sup>124</sup>.
			if z[31]&0xf0 != 0 {
				counter++
				continue
			}

			// 7.6. Define `s[t,j]` as `z[t,j]` with 4 high bits set to `mask[t]` bits.
			s[t][j] = z
			s[t][j][31] |= mask[t]
		}

		// 8. Set low 4 bits of `counter` to top 4 bits of `e0`.
		counterByte := byte(counter & 0xff)
		e0[31] |= ((counterByte << 4) & 0xf0)

		// 9. Return the borromean ring signature: `{e,s[t,j]}`: `n*m+1` 32-byte elements
		brs = new(borromeanRingSignature)
		brs.e = e0
		brs.s = s

		break
	}
	return brs, nil
}

// Inputs:
// 1. `msg`: the 32-byte string being verified.
// 2. `n`: number of rings.
// 3. `m`: number of signatures in each ring.
// 4. `{P[i,j]}`: `n*m` public keys, [points](data.md#public-key) on the elliptic curve.
// 5. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](data.md#borromean-ring-signature), `n*m+1` 32-byte elements.
func (brs *borromeanRingSignature) verify(
	msg [32]byte,
	pubkeys [][]Point,
) error {
	n := len(pubkeys)
	if n < 1 {
		return fmt.Errorf("n is 0")
	}
	m := len(pubkeys[0])
	if m < 1 {
		return fmt.Errorf("m is 0")
	}
	if len(brs.s) != n {
		return fmt.Errorf("number of s values %d does not match number of rings %d", len(brs.s), n)
	}

	// 1. Define `E` to be an empty binary string.
	E := hasher512()

	// 2. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
	cnt := byte(brs.e[31] >> 4)

	// 3. Set top 4 bits of `e0` to zero.
	e0 := brs.e
	e0[31] &= 0x0f

	// 4. For `t` from `0` to `n-1` (each ring):
	for t := 0; t < n; t++ {
		if len(brs.s[t]) != m {
			return fmt.Errorf("number of s values (%d) in ring %d does not match m (%d)", len(brs.s[t]), t, m)
		}
		if len(pubkeys[t]) != m {
			return fmt.Errorf("number of pubkeys (%d) in ring %d does not match m (%d)", len(pubkeys[t]), t, m)
		}

		// 4.1. Let `e[t,0] = e0`.
		e := e0

		// 4.2. For `i` from `0` to `m-1`:
		for i := 0; i < m; i++ {
			// 4.2.1. Calculate `z[t,i]` as `s[t,i]` with the most significant 4 bits set to zero.
			z := brs.s[t][i]
			z[31] &= 0x0f

			// 4.2.2. Calculate `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
			w := brs.s[t][i][31] & 0xf0

			// 4.2.3. Let `i’ = i+1 mod m`.
			i1 := (i + 1) % m

			// 4.2.4. Calculate point `R[t,i’] = z[t,i]·G - e[t,i]·P[t,i]` and encode it as a 32-byte [public key](data.md#public-key). Use `e0` instead of `e[t,0]` in each ring.
			R := multiplyAndAddPoint(negateScalar(e), pubkeys[t][i], z)

			// 4.2.5. Calculate `e[t,i’] = SHA3-512(R[t,i’] || msg || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
			// 4.2.6. Interpret `e[t,i’]` as a little-endian integer reduced modulo subgroup order `L`.
			e = computeInnerE(cnt, &R, msg[:], uint64(t), uint64(i1), w)
		}

		// 4.3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
		E.Write(e[:])
	}

	// 5. Calculate `e’ = SHA3-512(E)` and interpret it as a little-endian integer reduced modulo subgroup order `L`, and then encoded as a little-endian 32-byte integer.
	var e1hash [64]byte
	E.Sum(e1hash[:0])
	e1 := reducedScalar(e1hash)

	// 6. Return `true` if `e’` equals to `e0`. Otherwise, return `false`.
	if !constTimeEqual(e1[:], e0[:]) {
		return fmt.Errorf("borromean ringsig unverified")
	}
	return nil
}

// Inputs:
// 1. `msg`: the 32-byte string to be signed.
// 2. `n`: number of rings.
// 3. `m`: number of signatures in each ring.
// 4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
// 5. `{p[i]}`: the list of `n` scalars representing private keys.
// 6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·G`.
// 7. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](data.md#borromean-ring-signature), `n·m+1` 32-byte elements.
func (brs *borromeanRingSignature) recoverPayload(
	msg [32]byte,
	pubkeys [][]Point,
	privkeys []Scalar,
	indexes []int,
) ([][32]byte, error) {
	n := len(pubkeys)
	if n < 1 {
		return nil, fmt.Errorf("number of rings cannot be less than 1")
	}
	m := len(pubkeys[0])
	if m < 1 {
		return nil, fmt.Errorf("number of signatures per ring cannot be less than 1")
	}
	if len(privkeys) != n {
		return nil, fmt.Errorf("number of secret keys must equal number of rings")
	}
	if len(indexes) != n {
		return nil, fmt.Errorf("number of secret indexes must equal number of rings")
	}
	if len(brs.s) != n {
		return nil, fmt.Errorf("number of brs.s lists must be n")
	}

	payload := make([][32]byte, n*m)

	// 1. Define `E` to be an empty binary string.
	E := hasher512()

	// 2. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
	cnt := byte(brs.e[31] >> 4)

	// 3. Let `counter` integer equal `cnt`.
	counter := uint64(cnt)

	// 4. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = SHAKE256(counter || msg || {p[i]} || {j[i]} || {P[i,j]}, 8·(32·n·m))`, where:
	//     * `counter` is encoded as a 64-bit little-endian integer,
	//     * private keys `{p[i]}` are encoded as concatenation of 256-bit little-endian integers,
	//     * secret indexes `{j[i]}` are encoded as concatenation of 64-bit little-endian integers,
	//     * points `{P[i]}` are encoded as concatenation of [public keys](data.md#public-key).
	overlay := shake256(uint64le(counter), msg[:])
	for i := 0; i < n; i++ {
		overlay.Write(privkeys[i][:])
	}
	for i := 0; i < n; i++ {
		overlay.Write(uint64le(uint64(indexes[i])))
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			pBytes := encodePoint(&pubkeys[i][j])
			overlay.Write(pBytes[:])
		}
	}

	// 5. Set top 4 bits of `e0` to zero.
	e0 := brs.e
	e0[31] &= 0x0f

	// 6. For `t` from `0` to `n-1` (each ring):
	for t := 0; t < n; t++ {
		if len(brs.s[t]) != m {
			return nil, fmt.Errorf("number of elements in brs.s[%d] must be m=%d", t, m)
		}
		if len(pubkeys[t]) != m {
			return nil, fmt.Errorf("number of pubkeys[%d] must be m=%d", t, m)
		}

		// 6.1. Let `e[t,0] = e0`.
		e := e0

		// 6.2. For `i` from `0` to `m-1`:
		for i := 0; i < m; i++ {
			// 6.2.1. Calculate `z[t,i]` as `s[t,i]` with the most significant 4 bits set to zero.
			z := brs.s[t][i]
			z[31] &= 0x0f

			// 6.2.2. Calculate `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
			w := brs.s[t][i][31] & 0xf0

			var o [32]byte
			overlay.Read(o[:]) // overlay[m*t+i]

			// 6.2.3. If `i` is equal to `j[t]`:
			if i == indexes[t] {
				// 6.2.3.1. Calculate `k[t] = z[t,i] - p[t]·e[t,i] mod L`.
				k := multiplyAndAddScalars(negateScalar(e), privkeys[t], z)

				// 6.2.3.2. Set top 4 bits of `k[t]` to the top 4 bits of `w[t,i]`: `k[t][31] |= w[t,i]`.
				k[31] |= w

				// 6.2.3.3. Set `payload[m·t + i] = o[m·t + i] XOR k[t]`.
				payload[m*t+i] = xor256(o[:], k[:])

				// 6.2.4. If `i` is not equal to `j[t]`:
			} else {
				// 6.2.4.1. Set `payload[m·t + i] = o[m·t + i] XOR s[t,i]`.
				payload[m*t+i] = xor256(o[:], brs.s[t][i][:])
			}

			// 6.2.5. Let `i’ = i+1 mod m`.
			i1 := (i + 1) % m

			// 6.2.6. Calculate point `R[t,i’] = z[t,i]·G - e[t,i]·P[t,i]` and encode it as a 32-byte [public key](data.md#public-key). Use `e0` instead of `e[t,0]` in each ring.
			R := multiplyAndAddPoint(negateScalar(e), pubkeys[t][i], z)

			// 6.2.7. Calculate `e[t,i’] = SHA3-512(R[t,i’] || msg || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
			// 6.2.8. Interpret `e[t,i’]` as a little-endian integer reduced modulo subgroup order `L`.
			e = computeInnerE(cnt, &R, msg[:], uint64(t), uint64(i1), w)
		}

		// 6.3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
		E.Write(e[:])
	}

	// 7. Calculate `e’ = SHA3-512(E)` and interpret it as a little-endian integer reduced modulo subgroup order `L`, and then encoded as a little-endian 32-byte integer.
	var e1hash [64]byte
	E.Sum(e1hash[:0])
	e1 := reducedScalar(e1hash)

	// 8. Return `payload` if `e’` equals to `e0`. Otherwise, return `nil`.
	if !constTimeEqual(e1[:], e0[:]) {
		return nil, fmt.Errorf("borromean ring signature verification failed")
	}
	return payload, nil
}

// Calculates `e[t,i’] = SHA3-512(cnt, R[t,i’] || msg || t || i’ || w[t,i])`
// Interpret `e[t,i’]` as a little-endian integer reduced modulo `L`.
func computeInnerE(cnt byte, R *Point, msg []byte, t uint64, i uint64, w byte) Scalar {
	Renc := encodePoint(R)
	return reducedScalar(hash512([]byte{cnt}, Renc[:], msg, uint64le(t), uint64le(i), []byte{w}))
}

func (brs *borromeanRingSignature) String() string {
	outerS := make([]string, 0, len(brs.s))
	for _, slist := range brs.s {
		innerS := make([]string, 0, len(slist))
		for _, s := range slist {
			innerS = append(innerS, hex.EncodeToString(s[:]))
		}
		outerS = append(outerS, fmt.Sprintf("[%s]", strings.Join(innerS, ", ")))
	}
	return fmt.Sprintf("{e: %x, s: [%s]}", brs.e[:], strings.Join(outerS, ", "))
}
