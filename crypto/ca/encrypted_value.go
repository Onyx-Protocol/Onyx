package ca

import (
	"encoding/binary"

	"chain/crypto/ed25519/ecmath"
)

type EncryptedValue struct {
	ev [8]byte
	ef [32]byte
}

// EncryptValue encrypts value and blinding factor f
// using encryption key vek to the output buffer evef.
func EncryptValue(vc *ValueCommitment, value uint64, f ecmath.Scalar, vek ValueKey) *EncryptedValue {
	evef := new(EncryptedValue)

	// 1. Expand the encryption key: `ek = StreamHash("EV", {vek, VC}, 40)`.
	ekhash := streamHash("ChainCA.EV", vek, (*PointPair)(vc).Bytes())
	ekhash.Read(evef.ev[:])
	ekhash.Read(evef.ef[:])

	// 2. Encrypt the value using the first 8 bytes: `ev = value XOR ek[0,8]`.
	xorSlices(uint64le(value), evef.ev[:], evef.ev[:])

	// 3. Encrypt the value blinding factor using the last 32 bytes: `ef = f XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	xorSlices(f[:], evef.ef[:], evef.ef[:])

	return evef
}

// DecryptValue decrypts evef using key vek and verifies it using value commitment vc and asset ID commitment ac.
func (evef *EncryptedValue) Decrypt(vc *ValueCommitment, ac *AssetCommitment, vek ValueKey) (value uint64, f ecmath.Scalar, ok bool) {
	// 1. Expand the encryption key: `ek = StreamHash("EV", {vek, VC}, 40)`.
	ekhash := streamHash("ChainCA.EV", vek, vc.Bytes())

	var vbytes [8]byte
	ekhash.Read(vbytes[:])
	ekhash.Read(f[:])

	// 2. Decrypt the value using the first 8 bytes: `value = ev XOR ek[0,8]`.
	xorSlices(evef.ev[:], vbytes[:], vbytes[:])
	value = binary.LittleEndian.Uint64(vbytes[:])

	// 3. Decrypt the value blinding factor using the last 32 bytes: `f = ef XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	xorSlices(evef.ef[:], f[:], f[:])

	// 4. [Create blinded value commitment](#create-blinded-value-commitment) `VC’` using `AC`, `value` and the raw blinding factor `f` (instead of `vek`).
	vc2 := createRawValueCommitment(value, ac, &f)

	// 5. Verify that `VC’` equals `VC`. If not, halt and return `nil`.
	if !(*PointPair)(vc).ConstTimeEqual((*PointPair)(vc2)) {
		return 0, ecmath.Zero, false
	}

	// 6. Return `(value, f)`.
	return value, f, true
}
