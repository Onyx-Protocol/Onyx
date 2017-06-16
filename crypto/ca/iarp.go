package ca

import (
	"chain/crypto/ed25519/ecmath"
)

// AssetIssuanceKeyTuple interface allows passing:
// * issuance key specs during creation,
// * issuance choices during validation
// without the need to tediously reformat them to a temporary data structure.
type AssetIssuanceKeyTuple interface {
	AssetID() *AssetID
	IssuanceKey() IssuanceKey
}

type IssuanceAssetRangeProof interface {
	// TODO: add Reader/Writer interfaces
	Validate(ac *AssetCommitment) bool
}

type NonconfidentialIARP struct {
	AssetID AssetID
}

type ConfidentialIARP struct {
	TracingPoint  ecmath.Point
	IssuanceProof *OlegZKP
}

func CreateNonconfidentialIARP(assetID AssetID) *NonconfidentialIARP {
	return &NonconfidentialIARP{AssetID: assetID}
}

func (iarp *NonconfidentialIARP) Validate(ac *AssetCommitment) bool {
	ac2 := PointPair{ecmath.Point(CreateAssetPoint(iarp.AssetID)), ecmath.ZeroPoint}
	return (*PointPair)(ac).ConstTimeEqual(&ac2)
}

func CreateConfidentialIARP(
	ac *AssetCommitment,
	c *ecmath.Scalar,
	issuanceKeyTuples []AssetIssuanceKeyTuple,
	nonce []byte,
	message []byte,
	i int,
	y ecmath.Scalar,
) *ConfidentialIARP {

	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP", AC, uint64le(n),
	//                            a[0], ..., a[n-1],
	//                            Y[0], ..., Y[n-1],
	//                            nonce, message)

	// 2. Calculate marker point `M`:
	//         M = PointHash("IARP.M", basehash)

	// 3. Calculate the tracing point: `T = y·M`.
	// 4. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("msg", basehash, M, T)
	// 5. Calculate [asset ID points](#asset-id-point) for each `{a[i]}`:
	//         A[i] = PointHash("AssetID", a[i])
	// 6. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("h", msghash)
	// 7. Create [OLEG-ZKP](#oleg-zkp) with the following parameters:
	//     * `msghash`, the message hash to be signed.
	//     * `l = 2`, number of secrets.
	//     * `m = 3`, number of statements.
	//     * `{x[k]} = {c,y}`, secret scalars — blinding factor and an issuance key.
	//     * Statement sets (`i=0..n-1`):
	//             f[i,0](c,y) = (c + h·y)·G
	//             f[i,1](c,y) = c·J
	//             f[i,2](c,y) = y·M
	//     * Commitments (`i=0..n-1`):
	//             F[i,0] = H - A[i] + h·Y[i]
	//             F[i,1] = AC.C
	//             F[i,2] = T
	//     * `î`: the index of the non-forged item in a ring.
	// 12. Return [issuance asset range proof](#issuance-asset-range-proof) consisting of:
	//     * issuance keys `{Y[i]}`,
	//     * tracing point `T`,
	//     * OLEG-ZKP `e0,{s[i,k]}`.

	return &ConfidentialIARP{}
}
