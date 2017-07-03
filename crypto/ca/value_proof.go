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

	// 1. Compute a message hash to be signed:
	//         h = Hash256("AssetIDProof", {assetid, AC, message})
	h := hash256("ChainCA.AssetIDProof", assetID[:], (*PointPair)(ac).Bytes(), message)

	// 2. [Create excess commitment](#create-excess-commitment) `E` using scalar `c` and message `h`.
	e := CreateExcessCommitment(c, h[:])

	// 4. Return Schnorr signature extracted from the excess commitment (last 64 bytes):
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
	Q1 := *ac
	Q1[0].Sub(&Q1[0], (*ecmath.Point)(&a))

	//     3. [Validate excess commitment](#validate-excess-commitment) `Q || p || h`.
	var e ExcessCommitment
	e.QC = PointPair(Q1)
	e.setSignatureBytes(proof)
	e.msg = h[:]
	return e.Validate()
}

// ValueProofSize is a size of the value proof in bytes
const ValueProofSize = 128

type ValueProof [ValueProofSize]byte

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
) *ValueProof {
	// 1. Compute a message hash to be signed:
	//         h = Hash256("ValueProof", {assetid, uint64le(value), AC, VC, message})
	h := hash256("ChainCA.ValueProof",
		assetID[:],
		uint64le(value),
		(*PointPair)(ac).Bytes(),
		(*PointPair)(vc).Bytes(),
		message)

	// 2. [Create excess commitment](#create-excess-commitment) `E1` using scalar `c` and message `h`.
	e1 := CreateExcessCommitment(c, h[:])

	// 3. [Create excess commitment](#create-excess-commitment) `E2` using scalar `f` and message `h`.
	e2 := CreateExcessCommitment(f, h[:])

	// 4. Return concatenation of Schnorr signatures extracted from excess commitments (last 64 bytes from the each excess commitment):
	//         vp = E1[64:128] || E2[64:128]
	var result [ValueProofSize]byte
	copy(result[:64], e1.signatureBytes())
	copy(result[64:], e2.signatureBytes())

	return (*ValueProof)(&result)
}

// ValidateValueProof checks if a given proof vp actually proves that commitments
// ac and vc commit to a given assetID and value.
func (vp *ValueProof) Validate(
	assetID AssetID,
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	message []byte,
) bool {
	// 2. Compute a message hash to be signed:
	//         h = Hash256("ValueProof", {assetid, uint64le(value), AC, VC, message})
	h := hash256("ChainCA.ValueProof",
		assetID[:],
		uint64le(value),
		(*PointPair)(ac).Bytes(),
		(*PointPair)(vc).Bytes(),
		message)

	// 3. Compute [asset ID point](#asset-id-point): `A = PointHash("AssetID", assetID)`.
	a := CreateAssetPoint(&assetID)

	// 4. Subtract `A` from the first point of `AC` and leave second point unmodified:
	//         Q1 = AC - (A,O)
	Q1 := *ac
	Q1[0].Sub(&Q1[0], (*ecmath.Point)(&a))

	// 5. Scalar-multiply `AC` by `value` and subtract the resulting pair from `VC`:
	//         Q2 = VC - value·AC
	v := (&ecmath.Scalar{}).SetUint64(value)

	Q2 := *(*PointPair)(vc)
	tmp := *(*PointPair)(ac)
	tmp.ScMul(&tmp, v)
	Q2.Sub(&Q2, &tmp)

	// 6. [Validate excess commitment](#validate-excess-commitment) `Q1 || vp[0:64] || h`.
	// 7. [Validate excess commitment](#validate-excess-commitment) `Q2 || vp[64:128] || h`.
	var e1, e2 ExcessCommitment

	e1.QC = PointPair(Q1)
	e1.setSignatureBytes(vp[0:64])
	e1.msg = h[:]

	e2.QC = PointPair(Q2)
	e2.setSignatureBytes(vp[64:128])
	e2.msg = h[:]

	return e1.Validate() && e2.Validate()
}
