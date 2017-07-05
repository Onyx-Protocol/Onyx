package ca

import "chain/crypto/ed25519/ecmath"

type (
	// AmountProof proves that a value commitment is for a specific
	// amount (disregarding the asset type).
	AmountProof []byte

	// ValueProof proves that a value commitment is for a specific value
	// (amount+asset).
	ValueProof struct {
		asset  AssetProof
		amount AmountProof
	}
)

// CreateAmountProof creates a proof that a specific numeric value is
// commited by a given value commitment.
func CreateAmountProof(
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	f ecmath.Scalar,
	message []byte,
) AmountProof {
	if f.Equal(&ecmath.Zero) {
		return AmountProof{}
	}
	h := hash256("ChainCA.AmountProof", uint64le(value), (*PointPair)(ac).Bytes(), (*PointPair)(vc).Bytes(), message)
	e := CreateExcessCommitment(f, h[:])
	return AmountProof(e.signatureBytes())
}

// ValidateAmountProof checks a proof that a specific numeric value is
// commited by a given value commitment.
func (p AmountProof) Validate(
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	message []byte,
) bool {
	if !(len(p) == 0 || len(p) == 64) {
		return false
	}

	v := (&ecmath.Scalar{}).SetUint64(value)

	vcPrime := *(*PointPair)(ac)
	vcPrime.ScMul(&vcPrime, v)

	if len(p) == 0 {
		return (*PointPair)(vc).ConstTimeEqual(&vcPrime)
	}

	h := hash256("ChainCA.AmountProof", uint64le(value), ac.Bytes(), vc.Bytes(), message)
	Q := *(*PointPair)(vc)
	Q.Sub(&Q, &vcPrime)

	// 3. [Validate excess commitment](#validate-excess-commitment) `Q || p || h`.
	var e ExcessCommitment
	e.QC = PointPair(Q)
	e.setSignatureBytes(p)
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
) ValueProof {
	return ValueProof{
		asset:  ac.CreateProof(assetID, c, message),
		amount: CreateAmountProof(value, ac, vc, f, message),
	}
}

// ValidateValueProof checks if a given proof vp actually proves that commitments
// ac and vc commit to a given assetID and value.
func (p *ValueProof) Validate(
	assetID AssetID,
	value uint64,
	ac *AssetCommitment,
	vc *ValueCommitment,
	message []byte,
) bool {
	return p.asset.Validate(assetID, ac, message) && p.amount.Validate(value, ac, vc, message)
}
