package ca

import "chain/crypto/ed25519/ecmath"

// ValueCommitment is a point pair representing an ElGamal commitment
// to an asset ID commitment and an amount.
type ValueCommitment PointPair

// CreateValueCommitment creates a value commitment. Nil vek means
// make it nonblinded (and the returned scalar blinding factor is
// nil).
func CreateValueCommitment(value uint64, ac *AssetCommitment, vek ValueKey) (*ValueCommitment, *ecmath.Scalar) {
	var v ecmath.Scalar
	v.SetUint64(value)

	if vek == nil {
		// Non-blinded value commitment
		var vc PointPair
		vc.ScMul((*PointPair)(ac), &v)

		return (*ValueCommitment)(&vc), nil
	}
	// Blinded value commitment
	f := scalarHash("ChainCA.VC.f", uint64le(value), vek)
	var V, F, T ecmath.Point
	V.ScMulAdd(&ac.Point1, &v, &f) // V = value·H + f·G
	F.ScMul(&ac.Point2, &v)        // F = value·C
	T.ScMul(&J, &f)                // T = f·J
	F.Add(&F, &T)                  // F = value·C + f·J
	return (*ValueCommitment)(&PointPair{Point1: V, Point2: F}), &f
}
