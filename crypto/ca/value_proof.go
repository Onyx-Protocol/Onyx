package ca

import "chain/crypto/ed25519/ecmath"

// ValueProofSize is a size of the value proof in bytes
const ValueProofSize = 128

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
) []byte {
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
	return append(e1.SignatureBytes(), e2.SignatureBytes()...)
}

// func (vp *ValueProof) Validate(VC *ValueCommitment, assetID AssetID, value uint64, msg []byte) bool {
// 	if !(*ExcessCommitment)(vp).Validate() {
// 		return false
// 	}
// 	aPrime := CreateAssetPoint(assetID)
// 	var (
// 		v      ecmath.Scalar
// 		vPrime ecmath.Point
// 	)
// 	v.SetUint64(value)
// 	vPrime.ScMul((*ecmath.Point)(&aPrime), &v)
// 	vPrime.Add(&vPrime, &vp.QC.Point1)
// 	// TODO(bobg): always make both calls to ConstTimeEqual, to keep this function constant-time?
// 	if !vPrime.ConstTimeEqual(&VC.Point1) { // QG+V’ == VC.H?
// 		return false
// 	}
// 	return vp.QC.Point2.ConstTimeEqual(&VC.Point2) // QJ == VC.C?
// }
