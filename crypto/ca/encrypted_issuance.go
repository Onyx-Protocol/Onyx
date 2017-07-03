package ca

import "chain/crypto/ed25519/ecmath"

type EncryptedIssuance struct {
	ac   *AssetCommitment
	vc   *ValueCommitment
	iarp IssuanceAssetRangeProof
	vrp  *ValueRangeProof
}

// EncryptedIssuance encrypts an issuance. The issued asset is assetIDs[j].
func EncryptIssuance(
	value uint64,
	N uint64,
	j uint64,
	candidates []AssetIssuanceCandidate,
	y ecmath.Scalar,
	msg []byte,
	nonce [32]byte,
	aek AssetKey,
	vek ValueKey,
	idek DataKey,
) (ei *EncryptedIssuance, c, f ecmath.Scalar) {
	assetID := *candidates[j].AssetID()
	ac, cp := CreateAssetCommitment(assetID, aek)
	if cp == nil {
		c = ecmath.Zero
	} else {
		c = *cp
	}
	vc, fp := CreateValueCommitment(value, ac, vek)
	if fp == nil {
		f = ecmath.Zero
	} else {
		f = *fp
	}
	iarp := CreateConfidentialIARP(ac, c, candidates, nonce, msg, j, y)
	vrp := CreateValueRangeProof(ac, vc, N, value, nil, f, idek, vek, msg) // xxx msg? nil?
	return &EncryptedIssuance{ac: ac, vc: vc, iarp: iarp, vrp: vrp}, c, f
}
