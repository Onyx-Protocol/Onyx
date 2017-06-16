package ca

import "chain/crypto/ed25519/ecmath"

// ValueCommitment is a point pair representing an ElGamal commitment
// to an asset ID commitment and an amount.
type ValueCommitment PointPair

// CreateValueCommitment creates a value commitment. Nil vek means
// make it nonblinded (and the returned scalar blinding factor is
// nil).
func CreateValueCommitment(value uint64, ac *AssetCommitment, vek ValueKey) (*ValueCommitment, *ecmath.Scalar) {
	if vek == nil {
		v := (&ecmath.Scalar{}).SetUint64(value)

		// Non-blinded value commitment
		var vc PointPair
		vc.ScMul((*PointPair)(ac), v)

		return (*ValueCommitment)(&vc), nil
	}
	// Blinded value commitment
	f := scalarHash("ChainCA.VC.f", uint64le(value), vek)
	return createRawValueCommitment(value, ac, &f), &f
}

func createRawValueCommitment(value uint64, ac *AssetCommitment, f *ecmath.Scalar) *ValueCommitment {
	v := (&ecmath.Scalar{}).SetUint64(value)

	var V, F, T ecmath.Point
	V.ScMulAdd(&ac[0], v, f) // V = value·H + f·G
	F.ScMul(&ac[1], v)       // F = value·C
	T.ScMul(&J, f)           // T = f·J
	F.Add(&F, &T)            // F = value·C + f·J
	return (*ValueCommitment)(&PointPair{V, F})
}

// xxx make sure the signature of this function aligns with the spec
func ValidateValueCommitmentsBalance(inputs, outputs []*ValueCommitment, excesses []*ExcessCommitment) bool {
	for _, excess := range excesses {
		if !excess.Validate() {
			return false
		}
	}

	Ti := ZeroPointPair
	for _, inp := range inputs {
		Ti.Add(&Ti, (*PointPair)(inp))
	}

	Toq := ZeroPointPair
	for _, out := range outputs {
		Toq.Add(&Toq, (*PointPair)(out))
	}

	for _, excess := range excesses {
		Toq.Add(&Toq, &excess.QC)
	}

	return Ti.ConstTimeEqual(&Toq)
}

func (vc *ValueCommitment) V() *ecmath.Point { return &vc[0] }
func (vc *ValueCommitment) F() *ecmath.Point { return &vc[1] }

func (vc *ValueCommitment) Bytes() []byte {
	return (*PointPair)(vc).Bytes()
}
