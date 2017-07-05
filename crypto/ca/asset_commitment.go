package ca

import "chain/crypto/ed25519/ecmath"

// AssetCommitment is a point pair representing an ElGamal commitment to an AssetPoint.
type AssetCommitment PointPair

// AssetProof is a proof that an asset commitment is for a specific asset ID
type AssetProof []byte

// CreateAssetCommitment creates an asset commitment. Nil aek means
// make it nonblinded (and the returned scalar blinding factor is
// nil).
func CreateAssetCommitment(assetID AssetID, aek AssetKey) (*AssetCommitment, *ecmath.Scalar) {
	if aek == nil {
		A := ecmath.Point(CreateAssetPoint(&assetID))
		return &AssetCommitment{A, ecmath.ZeroPoint}, nil
	}
	c := scalarHash("ChainCA.AC.c", assetID[:], aek)
	return createRawAssetCommitment(assetID, &c), &c
}

func createRawAssetCommitment(assetID AssetID, c *ecmath.Scalar) *AssetCommitment {
	A := ecmath.Point(CreateAssetPoint(&assetID))
	var H, C ecmath.Point
	H.ScMulAdd(&A, &ecmath.One, c)
	C.ScMul(&J, c)
	return &AssetCommitment{H, C}
}

func (ac *AssetCommitment) H() *ecmath.Point { return &ac[0] }
func (ac *AssetCommitment) C() *ecmath.Point { return &ac[1] }

func (ac *AssetCommitment) Bytes() []byte {
	return (*PointPair)(ac).Bytes()
}

func (ac *AssetCommitment) Validate(assetID AssetID, aek AssetKey) bool {
	ac2, _ := CreateAssetCommitment(assetID, aek)
	return (*PointPair)(ac).ConstTimeEqual((*PointPair)(ac2))
}

func (ac *AssetCommitment) CreateProof(assetID AssetID, c ecmath.Scalar, msg []byte) AssetProof {
	if c.Equal(&ecmath.Zero) {
		return AssetProof{}
	}
	h := hash256("ChainCA.AssetIDProof", assetID[:], ac.Bytes(), msg)
	e := CreateExcessCommitment(c, h[:])
	return AssetProof(e.signatureBytes())
}

func (p AssetProof) Validate(assetID AssetID, ac *AssetCommitment, msg []byte) bool {
	// 1. If `p` is not an empty string or a 64-byte string, return `false`.
	if !(len(p) == 0 || len(p) == 64) {
		return false
	}
	// 2. Compute [asset ID point](#asset-id-point): `A = PointHash("AssetID", assetID)`.
	a := CreateAssetPoint(&assetID)

	// 3. If `p` is an empty string:
	//     1. Return `true` if `AC` equals `(A,O)`, return `false` otherwise.
	if len(p) == 0 {
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
	h := hash256("ChainCA.AssetIDProof", assetID[:], ac.Bytes(), msg)
	//     2. Subtract `A` from the first point of `AC` and leave second point unmodified:
	//             Q = AC - (A,O)
	Q := *ac
	Q[0].Sub(&Q[0], (*ecmath.Point)(&a))

	//     3. [Validate excess commitment](#validate-excess-commitment) `Q || p || h`.
	var e ExcessCommitment
	e.QC = PointPair(Q)
	e.setSignatureBytes(p)
	e.msg = h[:]
	return e.Validate()
}
