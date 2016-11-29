package ca

import (
	"fmt"
	"io"
	"math"

	"chain-stealth/encoding/blockchain"
)

type ValueRangeProof struct {
	N    uint8 // number of bits
	exp  uint8
	vmin uint64
	D    []Point // N/2-1 digit pedersen commitments
	brs  *borromeanRingSignature
}

func CreateValueRangeProof(
	H AssetCommitment,
	V ValueCommitment,
	evef EncryptedValue,
	N uint8, // the number of bits to be blinded.
	value uint64, // the 64-bit amount being encrypted and blinded.
	pt [][32]byte, // plaintext payload string consisting of `2·N - 1` 32-byte elements.
	f Scalar, // the value blinding factor
	rek RecordKey, // record encryption key
) (*ValueRangeProof, error) {
	// 1. Check that `N` belongs to the set `{8,16,32,48,64}`; if not, halt and return nil.
	if !(N == 8 || N == 16 || N == 32 || N == 48 || N == 64) {
		return nil, fmt.Errorf("Number of bits N must be in the set {8,16,32,48,64}")
	}
	// 2. Check that `value` is less than `2^N`; if not, halt and return nil.
	// This is done by the type system via uint64.

	// Check the length of the plaintext
	if len(pt) != int(2*N-1) {
		return nil, fmt.Errorf("Number of plaintext chunks must be 2*N-1, where N is number of bits")
	}
	// 3. Define `vmin = 0`.
	vmin := uint64(0)

	// 4. Define `exp = 0`.
	exp := uint8(0)

	// 5. Define `base = 4`.
	// 6. Calculate payload encryption key unique to this payload and the value: `pek = SHA3-256(0xec || rek || f || V)`.
	pek := hash256([]byte{0xec}, rek[:], f[:], V.Bytes())

	// 7. Calculate the message to sign: `msg = SHA3-256(H’ || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
	msg := hash256(
		H.Bytes(),
		V.Bytes(),
		uint64le(uint64(N)),
		uint64le(uint64(exp)),
		uint64le(vmin),
		evef.Value[:],
		evef.BlindingFactor[:],
	)

	// 8. Let number of digits `n = N/2`.
	n := N / 2

	// 9. [Encrypt the payload](#encrypt-payload) using `pek` as a key and `2·N-1` 32-byte plaintext elements to get `2·N` 32-byte ciphertext elements: `{ct[v]} = EncryptPayload({pt[v]}, pek)`.
	ct := EncryptPayload(pt, pek)

	// 10. Calculate 64-byte digit blinding factors for all but last digit: `{b[t]} = SHAKE256(0xbf || msg || f, 8·64·(n-1))`.
	b := make([]Scalar, n) // allocate array for the last digit too
	bshaker := shake256([]byte{0xbf}, msg[:], f[:])
	bsum := Scalar{}
	for i := uint8(0); i < n-1; i++ {
		bf := [64]byte{}
		bshaker.Read(bf[:])
		// 11. Interpret each 64-byte `b[t]` (`t` from 0 to `n-2`) is interpreted as a little-endian integer and reduce modulo `L` to a 32-byte scalar.
		b[i] = reducedScalar(bf)
		bsum.Add(&b[i])
	}
	// 12. Calculate the last digit blinding factor: `b[n-1] = f - ∑b[t] mod L`, where `t` is from 0 to `n-2`.
	b[n-1] = subScalars(f, bsum)

	P := make([][]Point, n)
	D := make([]Point, n)
	j := make([]int, n)

	// 13. For `t` from `0` to `n-1` (each digit):
	coeffBase := 1
	for t := uint8(0); t < n; t++ {
		// 13.1. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
		digit := value & (0x03 << uint(2*t))

		// 13.2. Calculate `D[t] = digit[t]·H + b[t]·G`.
		D[t] = multiplyAndAddPoint(scalarFromUint64(digit), Point(H), b[t])

		// 13.3. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
		j[t] = int(digit >> (2 * t))

		// 13.4. For `i` from `0` to `base-1` (each digit’s value):
		// 13.4.1. Calculate point `P[t,i] = D[t] - i·(base^t)·H’`.
		P[t] = calcDigitPoints(n, coeffBase, &H, &D[t])

		coeffBase *= 4
	}

	// 14. [Create Borromean Ring Signature](#create-borromean-ring-signature) `brs` with the following inputs:
	//     1. `msg` as the message to sign.
	//     2. `n`: number of rings.
	//     3. `m = base`: number of signatures per ring.
	//     4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
	//     5. `{b[i]}`: the list of `n` blinding factors as private keys.
	//     6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == b[i]·G`.
	//     7. `{r[v]} = {ct[v]}`: random string consisting of `n·m` 32-byte ciphertext elements.
	brs, err := createBorromeanRingSignature(msg, P, b, j, ct)

	// 15. If failed to create borromean ring signature `brs`, return nil.
	// The chance of this happening is below 1 in 2<sup>124</sup>.
	// In case of failure, retry [creating blinded value commitments](#create-blinded-value-commitments) with incremented counter.
	// This would yield a new blinding factor `f` that will produce different digit blinding keys in this algorithm.
	if err != nil {
		return nil, err
	}

	// 16. Return the [value range proof](#value-range-proof):
	//     * `N`:  number of blinded bits (equals to `2·n`),
	//     * `exp`: exponent (zero),
	//     * `vmin`: minimum value (zero),
	//     * `{D[t]}`: `n-1` digit commitments encoded as [public keys](data.md#public-key) (excluding the last digit commitment),
	//     * `{e,s[t,j]}`: `1 + n·4` 32-byte elements representing a [borromean ring signature](data.md#borromean-ring-signature),
	vrp := &ValueRangeProof{
		N:    N,
		exp:  exp,
		vmin: vmin,
		D:    D[:len(D)-1],
		brs:  brs,
	}
	return vrp, nil
}

func (vrp *ValueRangeProof) Verify(
	H AssetCommitment, // assumed to be verified
	V ValueCommitment, // to be verified in this function
	evef *EncryptedValue,
) error {
	// 1. Perform limit checks
	if !vrp.CheckLimits() {
		return fmt.Errorf("limits check failed")
	}

	// 2. Let `n = N/2`.
	n := vrp.N / 2

	if evef == nil {
		evef = &EncryptedValue{}
	}

	// 3. Calculate the message to verify: `msg = SHA3-256(H’ || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
	msg := hash256(H.Bytes(), V.Bytes(), uint64le(uint64(vrp.N)), uint64le(uint64(vrp.exp)), uint64le(vrp.vmin), evef.Value[:], evef.BlindingFactor[:])

	// 4. Calculate last digit commitment `D[m-1] = (10^(-exp))·(V - vmin·H’) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
	powerOf10, ok := powersOf10[int(-vrp.exp)] // 10^(-exp)
	if !ok {
		return fmt.Errorf("unavailable power of ten (%d)", int(-vrp.exp))
	}

	Dsum := ZeroPoint
	for _, D := range vrp.D {
		Dsum.add(&D)
	}

	vminH := multiplyPoint(scalarFromUint64(vrp.vmin), Point(H)) // vmin·H’
	Dlast := subPoints(Point(V), vminH)                          // V - vmin·H’
	Dlast = multiplyPoint(powerOf10, Dlast)                      // (10^(-exp))·(V - vmin·H’)
	Dlast.sub(&Dsum)                                             // (10^(-exp))·(V - vmin·H’) - ∑(D[t])

	P := make([][]Point, n)
	coeffBase := 1
	// 5. For `t` from `0` to `n-1` (each ring):
	for t := uint8(0); t < n; t++ {
		var D *Point
		if t == n-1 {
			D = &Dlast
		} else {
			D = &vrp.D[t]
		}

		//     1. Define `base = 4`.
		//     2. For `i` from `0` to `base-1` (each digit’s value):
		//         1. Calculate point `P[t,i] = D[t] - i·(base^t)·H’`.
		P[t] = calcDigitPoints(n, coeffBase, &H, D)

		coeffBase *= 4
	}

	// 6. [Verify Borromean Ring Signature](#verify-borromean-ring-signature) with the following inputs:
	//     1. `msg`: the 32-byte string being verified.
	//     2. `n`: number of rings.
	//     3. `m=base`: number of signatures in each ring.
	//     4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
	//     5. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](data.md#borromean-ring-signature), `n·m+1` 32-byte elements.
	// 7. Return `true` if verification succeeded, or `false` otherwise.
	return vrp.brs.verify(msg, P)
}

// Inputs
// 1. `H`: the [verified](#verify-asset-range-proof) [asset ID commitment](#asset-id-commitment).
// 2. `V`: the [value commitment](#value-commitment).
// 3. `(ev,ef)`: the [encrypted value](#encrypted-value) including its blinding factor.
// 4. Value range proof consisting of:
//     * `N`: the number of bits in blinded mantissa (8-bit integer, `N = 2·n`).
//     * `exp`: the decimal exponent (8-bit integer).
//     * `vmin`: the minimum amount (64-bit integer).
//     * `{D[t]}`: the list of `n-1` digit pedersen commitments encoded as [public keys](data.md#public-key).
//     * `{e0, s[i,j]...}`: the [borromean ring signature](#borromean-ring-signature) encoded as a sequence of `1 + 4·n` 32-byte integers.
// 5. `value`: the 64-bit amount being encrypted and blinded.
// 6. `f`: the [value blinding factor](#value-blinding-factor).
// 7. `rek`: the [record encryption key](#record-encryption-key).
func (vrp *ValueRangeProof) RecoverPayload(
	H AssetCommitment, // assumed to be verified
	V ValueCommitment, // to be verified in this function
	evef *EncryptedValue,
	value uint64, // the 64-bit amount being encrypted and blinded.
	f Scalar, // the value blinding factor
	rek RecordKey, // record encryption key
) (pt [][32]byte, err error) { // `{pt[i]}`: an array of 32-bytes of plaintext data if recovery succeeded, `nil` otherwise.

	// 1. Perform limit checks
	if !vrp.CheckLimits() {
		return pt, fmt.Errorf("value range proof did not pass limit checks")
	}

	// 2. Let `n = N/2`.
	n := vrp.N / 2

	if evef == nil {
		evef = &EncryptedValue{}
	}

	// 3. Calculate the message to verify: `msg = SHA3-256(H’ || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
	msg := hash256(H.Bytes(), V.Bytes(), uint64le(uint64(vrp.N)), uint64le(uint64(vrp.exp)), uint64le(vrp.vmin), evef.Value[:], evef.BlindingFactor[:])

	// 4. Calculate last digit commitment `D[m-1] = (10^(-exp))·(V - vmin·H’) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
	powerOf10, ok := powersOf10[int(-vrp.exp)] // 10^(-exp)
	if !ok {
		return pt, fmt.Errorf("value range proof has out of range exponent")
	}

	Dsum := ZeroPoint
	for _, D := range vrp.D {
		Dsum.add(&D)
	}

	vminH := multiplyPoint(scalarFromUint64(vrp.vmin), Point(H)) // vmin·H’
	Dlast := subPoints(Point(V), vminH)                          // V - vmin·H’
	Dlast = multiplyPoint(powerOf10, Dlast)                      // (10^(-exp))·(V - vmin·H’)
	Dlast.sub(&Dsum)                                             // (10^(-exp))·(V - vmin·H’) - ∑(D[t])

	// 5. Calculate 64-byte digit blinding factors for all but last digit: `{b[t]} = SHAKE256(0xbf || msg || f, 8·64·(n-1))`.
	b := make([]Scalar, n) // allocate array for the last digit too
	bshaker := shake256([]byte{0xbf}, msg[:], f[:])
	bsum := Scalar{}
	for i := uint8(0); i < n-1; i++ {
		bf := [64]byte{}
		bshaker.Read(bf[:])
		// 6. Interpret each 64-byte `b[t]` (`t` from 0 to `n-2`) is interpreted as a little-endian integer and reduce modulo `L` to a 32-byte scalar.
		b[i] = reducedScalar(bf)
		bsum.Add(&b[i])
	}
	// 7. Calculate the last digit blinding factor: `b[n-1] = f - ∑b[t] mod L`, where `t` is from 0 to `n-2`.
	b[n-1] = subScalars(f, bsum)

	P := make([][]Point, n)
	j := make([]int, n)

	// 8. For `t` from `0` to `n-1` (each digit):
	coeffBase := 1
	for t := uint8(0); t < n; t++ {
		//     1. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
		digit := value & (0x03 << uint(2*t))

		//     2. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
		j[t] = int(digit >> (2 * t))

		var D *Point
		if t == n-1 {
			D = &Dlast
		} else {
			D = &vrp.D[t]
		}
		P[t] = make([]Point, 4)
		//     3. Define `base = 4`.
		//     4. For `i` from `0` to `base-1` (each digit’s value):
		for i := 0; i < 4; i++ {
			//         1. Calculate point `P[t,i] = D[t] - i·(base^t)·H’`.
			P[t][i] = subPoints(*D, multiplyPoint(scalarFromUint64(uint64(i*coeffBase)), Point(H)))
		}
		coeffBase *= 4
	}

	// 9. [Recover Payload From Borromean Ring Signature](#recover-payload-from-borromean-ring-signature): compute an array of `2·N` 32-byte chunks `{ct[i]}` using the following inputs (halt and return `nil` if decryption fails):
	//     1. `msg`: the 32-byte string to be signed.
	//     2. `n=N/2`: number of rings.
	//     3. `m=base`: number of signatures in each ring.
	//     4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
	//     5. `{b[i]}`: the list of `n` blinding factors as private keys.
	//     6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == b[i]·G`.
	//     7. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
	ct, err := vrp.brs.recoverPayload(
		msg,
		P,
		b,
		j,
	)
	if err != nil {
		return pt, err
	}

	// 10. Derive payload encryption key unique to this payload and the value: `pek = SHA3-256(0xec || rek || f || V)`.
	pek := hash256([]byte{0xec}, rek[:], f[:], V.Bytes())

	// 11. [Decrypt payload](#decrypt-payload): compute an array of `2·N-1` 32-byte chunks: `{pt[i]} = DecryptPayload({ct[i]}, pek)`. If decryption fails, halt and return `nil`.
	pt, ok = DecryptPayload(ct, pek)
	if !ok {
		return pt, fmt.Errorf("failed to verify decrypted payload")
	}

	// 12. Return `{pt[i]}`, a plaintext array of `2·N-1` 32-byte elements.
	return pt, nil
}

func calcDigitPoints(n uint8, coeffBase int, H *AssetCommitment, D *Point) []Point {
	res := make([]Point, 4)
	for i := 0; i < 4; i++ {
		res[i] = subPoints(*D, multiplyPoint(scalarFromUint64(uint64(i*coeffBase)), Point(*H)))
	}
	return res
}

func (vrp *ValueRangeProof) CheckLimits() bool {
	// 1. Perform limit checks one by one. If any one fails, halt and return `false`:
	//     1. Check that `exp` is less or equal to 10.
	if vrp.exp > 10 {
		return false
	}

	//     2. Check that `vmin` is less than 2<sup>63</sup>.
	if vrp.vmin > math.MaxInt64 {
		return false
	}

	//     3. Check that `N` is divisible by 2.
	if vrp.N%2 != 0 {
		return false
	}

	//     4. Check that `N` is equal or less than 64.
	//     5. Check that `N + exp·4` is less or equal to 64.
	if 4*uint64(vrp.exp)+uint64(vrp.N) > 64 {
		return false
	}

	//     6. Check that `(10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
	//     7. Check that `vmin + (10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
	// TODO: review these checks and fix the bugs here.
	p := uint64(1)
	for i := uint8(0); i < vrp.exp; i++ {
		p *= 10
	}
	if p*2*uint64(vrp.N)+vrp.vmin-1 > math.MaxInt64 {
		return false
	}

	return true
}

func (vrp *ValueRangeProof) WriteTo(w io.Writer) error {
	b := []byte{vrp.N}
	_, err := w.Write(b)
	if err != nil {
		return err
	}
	b = []byte{vrp.exp}
	_, err = w.Write(b)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, vrp.vmin)
	if err != nil {
		return err
	}
	for _, d := range vrp.D {
		err = d.WriteTo(w)
		if err != nil {
			return err
		}
	}
	return vrp.brs.writeTo(w)
}

func (vrp *ValueRangeProof) ReadFrom(r io.Reader) error {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return err
	}
	vrp.N = b[0]

	_, err = io.ReadFull(r, b[:])
	if err != nil {
		return err
	}
	vrp.exp = b[0]

	vrp.vmin, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}

	n := vrp.N / 2
	vrp.D = make([]Point, n-1)
	for i := uint8(0); i < n-1; i++ {
		err = vrp.D[i].readFrom(r)
		if err != nil {
			return err
		}
	}

	vrp.brs = new(borromeanRingSignature)
	return vrp.brs.readFrom(r, int(n), 4)
}
