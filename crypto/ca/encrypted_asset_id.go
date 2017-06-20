package ca

import "chain/crypto/ed25519/ecmath"

type EncryptedAssetID struct {
	ea, ec [32]byte
}

// EncryptAssetID encrypts value and blinding factor f
// using encryption key vek to the output buffer evef.
func EncryptAssetID(ac *AssetCommitment, assetID AssetID, c ecmath.Scalar, aek AssetKey) *EncryptedAssetID {
	result := new(EncryptedAssetID)

	// 1. Expand the encryption key: `ek = StreamHash("EA", {aek, AC}, 40)`.
	ekhash := streamHash("ChainCA.EA", aek, ac.Bytes())
	ekhash.Read(result.ea[:])
	ekhash.Read(result.ec[:])

	// 2. Encrypt the asset ID using the first 32 bytes: `ea = assetID XOR ek[0,32]`.
	xorSlices(assetID[:], result.ea[:], result.ea[:])

	// 3. Encrypt the blinding factor using the second 32 bytes: `ec = c XOR ek[32,32]` where `c` is encoded as a 256-bit little-endian integer.
	xorSlices(c[:], result.ec[:], result.ec[:])

	return result
}

// DecryptAssetID decrypts eaec using key aek and verifies it using ID commitment ac.
func (eaec *EncryptedAssetID) Decrypt(ac *AssetCommitment, aek AssetKey) (assetID AssetID, c ecmath.Scalar, ok bool) {
	// 1. Expand the decryption key: `ek = StreamHash("EA", {aek, AC}, 40)`.
	var eka, ekc [32]byte
	ekhash := streamHash("ChainCA.EA", aek, ac.Bytes())
	ekhash.Read(eka[:])
	ekhash.Read(ekc[:])

	// 2. Decrypt the asset ID using the first 32 bytes: `assetID = ea XOR ek[0,32]`.
	xorSlices(eaec.ea[:], eka[:], assetID[:])

	// 3. Decrypt the blinding factor using the second 32 bytes: `c = ec XOR ek[32,32]`.
	xorSlices(eaec.ec[:], ekc[:], c[:])

	// 4. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment) `AC’` using `assetID` and the raw blinding factor `c` (instead of `aek`).
	ac2 := createRawAssetCommitment(assetID, &c)

	// 5. Verify that `AC’` equals `AC`. If not, halt and return `nil`.
	if !(*PointPair)(ac).ConstTimeEqual((*PointPair)(ac2)) {
		return AssetID{}, ecmath.Zero, false
	}

	// 6. Return `(assetID, c)`.
	return assetID, c, true
}
