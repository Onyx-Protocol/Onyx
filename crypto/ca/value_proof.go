package ca

import "chain/crypto/ed25519/ecmath"

// CreateAssetIDProof creates a proof of a specifc asset ID bound to
// the given asset ID commitment.
func CreateAssetIDProof(
	assetID AssetID,
	ac *AssetCommitment,
	c ecmath.Scalar,
	message []byte,
) []byte {
	if c.Equal(&ecmath.Zero) {
		return []byte{}
	}
	h := hash256("ChainCA.AssetIDProof", assetID[:], (*PointPair)(ac).Bytes(), message)
	e := CreateExcessCommitment(c, h[:])
	return e.signatureBytes()
}

// ValidateAssetIDProof checks if a given proof actually proves that commitment ac commits to a given assetID.
func ValidateAssetIDProof(
	assetID AssetID,
	ac *AssetCommitment,
	message []byte,
	proof []byte,
) bool {
	// 1. If `p` is not an empty string or a 64-byte string, return `false`.
	if !(len(proof) == 0 || len(proof) == 64) {
		return false
	}
	// 2. Compute [asset ID point](#asset-id-point): `A = PointHash("AssetID", assetID)`.
	a := CreateAssetPoint(&assetID)

	// 3. If `p` is an empty string:
	//     1. Return `true` if `AC` equals `(A,O)`, return `false` otherwise.
	if len(proof) == 0 {
		if !ac[0].ConstTimeEqual((*ecmath.Point)(&a)) {
			return false
		}
		if !ac[1].ConstTimeEqual(&ecmath.ZeroPoint) {
			return false
		}
		return true
	}
	// 4. If `p` is not an empty string:
	//     1. Compute a message hash to be signed:
	//             h = Hash256("AssetIDProof", {assetid, AC, message})
	h := hash256("ChainCA.AssetIDProof",
		assetID[:],
		(*PointPair)(ac).Bytes(),
		message)
	//     2. Subtract `A` from the first point of `AC` and leave second point unmodified:
	//             Q = AC - (A,O)
	Q := *ac
	Q[0].Sub(&Q[0], (*ecmath.Point)(&a))

	//     3. [Validate excess commitment](#validate-excess-commitment) `Q || p || h`.
	var e ExcessCommitment
	e.QC = PointPair(Q)
	e.setSignatureBytes(proof)
	e.msg = h[:]
	return e.Validate()
}

// CreateAmountProof creates a proof that a specific numeric value is
// commited by a given value commitment.
func CreateAmountProof(
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	f ecmath.Scalar,
	message []byte,
) []byte {
	if f.Equal(&ecmath.Zero) {
		return []byte{}
	}
	h := hash256("ChainCA.AmountProof", uint64le(value), (*PointPair)(ac).Bytes(), (*PointPair)(vc).Bytes(), message)
	e := CreateExcessCommitment(f, h[:])
	return e.signatureBytes()
}

// ValidateAmountProof checks a proof that a specific numeric value is
// commited by a given value commitment.
func ValidateAmountProof(
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	message []byte,
	proof []byte,
) bool {

	if !(len(proof) == 0 || len(proof) == 64) {
		return false
	}

	v := (&ecmath.Scalar{}).SetUint64(value)

	vcPrime := *(*PointPair)(ac)
	vcPrime.ScMul(&vcPrime, v)

	if len(proof) == 0 {
		return (*PointPair)(vc).ConstTimeEqual(&vcPrime)
	}

	h := hash256("ChainCA.AmountProof", uint64le(value), (*PointPair)(ac).Bytes(), (*PointPair)(vc).Bytes(), message)
	Q := *(*PointPair)(vc)
	Q.Sub(&Q, &vcPrime)

	// 3. [Validate excess commitment](#validate-excess-commitment) `Q || p || h`.
	var e ExcessCommitment
	e.QC = PointPair(Q)
	e.setSignatureBytes(proof)
	e.msg = h[:]
	return e.Validate()
}

// CreateValueProof creates a proof of a specifc asset ID and value bound to
// the given asset ID commitment and value commitment.
func CreateValueProof(
	assetID AssetID,
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	c ecmath.Scalar,
	f ecmath.Scalar,
	message []byte,
) [][]byte {

	return [][]byte{
		CreateAssetIDProof(assetID, ac, c, message),
		CreateAmountProof(value, ac, vc, f, message),
	}
}

// ValidateValueProof checks if a given proof vp actually proves that commitments
// ac and vc commit to a given assetID and value.
func ValidateValueProof(
	assetID AssetID,
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	message []byte,
	proof [][]byte,
) bool {
	if len(proof) != 2 {
		return false
	}
	return ValidateAssetIDProof(assetID, ac, message, proof[0]) &&
		ValidateAmountProof(value, ac, vc, message, proof[1])
}
