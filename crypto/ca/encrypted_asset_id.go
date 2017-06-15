package ca

import (
	"chain/crypto/ed25519/ecmath"
)

// EncryptedAssetIDSize is size in bytes of an encrypted asset ID and its blinding factor
const EncryptedAssetIDSize = 64

// EncryptAsset encrypts value and blinding factor f
// using encryption key vek to the output buffer evef.
func EncryptAsset(ac *AssetCommitment, assetID AssetID, c *ecmath.Scalar, aek AssetKey, eaec []byte) {
	if len(eaec) != EncryptedAssetIDSize {
		panic("Invalid buffer size for the encrypted asset ID: should have EncryptedAssetIDSize bytes.")
	}
	// 1. Expand the encryption key: `ek = StreamHash("EA", {aek, AC}, 40)`.
	ekhash := streamHash("ChainCA.EA", aek, (*PointPair)(ac).Bytes())
	ekhash.Read(eaec[:]) // reuse output buffer for the key

	// 2. Encrypt the asset ID using the first 32 bytes: `ea = assetID XOR ek[0,32]`.
	xorSlices(assetID[:], eaec[0:32], eaec[0:32])

	// 3. Encrypt the blinding factor using the second 32 bytes: `ec = c XOR ek[32,32]` where `c` is encoded as a 256-bit little-endian integer.
	xorSlices(c[:], eaec[32:64], eaec[32:64])
}

// DecryptAsset decrypts eaec using key aek and verifies it using ID commitment ac.
func DecryptAsset(eaec []byte, vc *ValueCommitment, ac *AssetCommitment, aek AssetKey) (assetID AssetID, c *ecmath.Scalar, ok bool) {
	if len(eaec) != EncryptedAssetIDSize {
		return AssetID{}, &ecmath.Zero, false
	}

	// 1. Expand the decryption key: `ek = StreamHash("EA", {aek, AC}, 40)`.
	var ek [EncryptedAssetIDSize]byte
	ekhash := streamHash("ChainCA.EA", aek, (*PointPair)(ac).Bytes())
	ekhash.Read(ek[:])

	// 2. Decrypt the asset ID using the first 32 bytes: `assetID = ea XOR ek[0,32]`.
	xorSlices(eaec[0:32], ek[0:32], assetID[:])

	// 3. Decrypt the blinding factor using the second 32 bytes: `c = ec XOR ek[32,32]`.
	cbuf := ecmath.Zero
	c = &cbuf
	xorSlices(eaec[32:], ek[32:], cbuf[:])

	// 4. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment) `AC’` using `assetID` and the raw blinding factor `c` (instead of `aek`).
	ac2 := createRawAssetCommitment(assetID, c)

	// 5. Verify that `AC’` equals `AC`. If not, halt and return `nil`.
	if !(*PointPair)(ac).ConstTimeEqual((*PointPair)(ac2)) {
		return AssetID{}, &ecmath.Zero, false
	}

	// 6. Return `(assetID, c)`.
	return assetID, c, true
}

// func (enc EncryptedAssetID) Decrypt(
// 	H AssetCommitment,
// 	aek AssetKey,
// ) (assetID AssetID, c Scalar, err error) {

// 	// 1. Expand the encryption key: `ek = SHA3-512(aek || H’)`, split the resulting hash in two halves.
// 	ek := hash512(aek[:], H.Bytes())

// 	// 2. Decrypt the asset ID using the first half: `assetID = ea XOR ek[0,32]`.
// 	assetID = AssetID(xor256(enc.AssetID[:], ek[0:32]))

// 	// 3. Decrypt the cumulative blinding factor using the second half: `c = ec XOR ek[32,32]`.
// 	c = xor256(enc.BlindingFactor[:], ek[32:64])

// 	// 4. Calculate `A` as a nonblinded asset ID commitment:
// 	//    an elliptic curve point `8*decode(SHA3-256(assetID))`.
// 	A := CreateNonblindedAssetCommitment(assetID)

// 	// 5. Calculate point `P = A + c*G` where `c` is interpreted as a little-endian 256-bit integer.
// 	P := multiplyAndAddPoint(one, Point(A), c)

// 	// 6. Verify that `P` equals target commitment `H’`. If not, halt and return `nil`.
// 	if encodePoint(&P) != encodePoint((*Point)(&H)) {
// 		return assetID, c, errors.New("Asset ID decryption failed")
// 	}

// 	// 7. Return `(assetID, c)`.
// 	return assetID, c, nil
// }
