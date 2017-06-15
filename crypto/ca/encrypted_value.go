package ca

import (
	"chain/crypto/ed25519/ecmath"
	"encoding/binary"
)

// EncryptedValueSize is size in bytes of an encrypted value and its blinding factor
const EncryptedValueSize = 40

// EncryptValue encrypts value and blinding factor f
// using encryption key vek to the output buffer evef.
func EncryptValue(vc *ValueCommitment, value uint64, f *ecmath.Scalar, vek ValueKey, evef []byte) {
	if len(evef) != EncryptedValueSize {
		panic("Invalid buffer size for the encrypted value: should have EncryptedValueSize bytes.")
	}

	// 1. Expand the encryption key: `ek = StreamHash("EV", {vek, VC}, 40)`.
	ekhash := streamHash("ChainCA.EV", vek, (*PointPair)(vc).Bytes())
	ekhash.Read(evef[:]) // reuse output buffer for the key

	// 2. Encrypt the value using the first 8 bytes: `ev = value XOR ek[0,8]`.
	xorSlices(uint64le(value), evef[0:8], evef[0:8])

	// 3. Encrypt the value blinding factor using the last 32 bytes: `ef = f XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	xorSlices(f[:], evef[8:40], evef[8:40])
}

// DecryptValue decrypts evef using key vek and verifies it using value commitment vc and asset ID commitment ac.
func DecryptValue(evef []byte, vc *ValueCommitment, ac *AssetCommitment, vek ValueKey) (value uint64, f *ecmath.Scalar, ok bool) {
	if len(evef) != EncryptedValueSize {
		return 0, &ecmath.Zero, false
	}

	// 1. Expand the encryption key: `ek = StreamHash("EV", {vek, VC}, 40)`.
	var ek [EncryptedValueSize]byte
	ekhash := streamHash("ChainCA.EV", vek, (*PointPair)(vc).Bytes())
	ekhash.Read(ek[:])

	// 2. Decrypt the value using the first 8 bytes: `value = ev XOR ek[0,8]`.
	var vbytes [8]byte
	xorSlices(evef[0:8], ek[0:8], vbytes[:])
	value = binary.LittleEndian.Uint64(vbytes[:])

	// 3. Decrypt the value blinding factor using the last 32 bytes: `f = ef XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	fbuf := ecmath.Zero
	f = &fbuf
	xorSlices(evef[8:], ek[8:], fbuf[:])

	// 4. [Create blinded value commitment](#create-blinded-value-commitment) `VC’` using `AC`, `value` and the raw blinding factor `f` (instead of `vek`).
	vc2 := createRawValueCommitment(value, ac, f)

	// 5. Verify that `VC’` equals `VC`. If not, halt and return `nil`.
	if !(*PointPair)(vc).ConstTimeEqual((*PointPair)(vc2)) {
		return 0, &ecmath.Zero, false
	}

	// 6. Return `(value, f)`.
	return value, f, true
}
