package ca

import "chain/crypto/ed25519/ecmath"

type ValueProof ExcessCommitment

func CreateValueProof(value uint64, c, f ecmath.Scalar, msg []byte) *ValueProof {
	var v ecmath.Scalar
	v.SetUint64(value)

	var q ecmath.Scalar
	q.MulAdd(&v, &c, &f)
	return (*ValueProof)(CreateExcessCommitment(q, msg))
}

func (vp *ValueProof) Validate(VC *ValueCommitment, assetID AssetID, value uint64, msg []byte) bool {
	if !(*ExcessCommitment)(vp).Validate(msg) {
		return false
	}
	aPrime := CreateAssetPoint(assetID)
	var (
		v      ecmath.Scalar
		vPrime ecmath.Point
	)
	v.SetUint64(value)
	vPrime.ScMul((*ecmath.Point)(&aPrime), &v)
	vPrime.Add(&vPrime, &vp.QC.Point1)
	// TODO(bobg): always make both calls to ConstTimeEqual, to keep this function constant-time?
	if !vPrime.ConstTimeEqual(&VC.Point1) { // QG+V’ == VC.H?
		return false
	}
	return vp.QC.Point2.ConstTimeEqual(&VC.Point2) // QJ == VC.C?
}
