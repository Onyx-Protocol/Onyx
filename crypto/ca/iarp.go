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
	c ecmath.Scalar,
	issuanceKeyTuples []AssetIssuanceKeyTuple,
	nonce []byte,
	message []byte,
	secretIndex uint64,
	y ecmath.Scalar,
) *ConfidentialIARP {

	n := uint64(len(issuanceKeyTuples))

	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                         {AC, nonce, message,
	//                         uint64le(n),
	//                         a[0],Y[0], ..., a[n-1],Y[n-1]})
	basehasher := hasher256("ChainCA.IARP.base",
		ac.Bytes(),
		nonce,
		message,
		uint64le(n))

	for _, tuple := range issuanceKeyTuples {
		basehasher.WriteItem(tuple.AssetID()[:])
		basehasher.WriteItem(tuple.IssuanceKey())
	}
	var basehash [32]byte
	basehasher.Sum(basehash[:0])

	// 2. Calculate marker point `M`:
	//         M = PointHash("IARP.M", {basehash})
	M := pointHash("ChainCA.IARP.M", basehash[:])

	// 3. Calculate the tracing point: `T = y·M`.
	var T ecmath.Point
	T.ScMul(&M, &y)

	// 4. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("IARP.msg", {basehash, M, T})
	msghash := hash256("ChainCA.IARP.msg", basehash[:], M.Bytes(), T.Bytes())

	// 5. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("IARP.h", {msghash})
	//h := scalarHash("ChainCA.IARP.h", msghash[:])

	//f := make([][]OlegZKPFunc, )
	for i := uint64(0); i < n; i++ {

	}

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
	ozkp := CreateOlegZKP(
		msghash[:],
		[]ecmath.Scalar{c, y},
		[]OlegZKPFunc{},
		[][]ecmath.Point{},
		secretIndex)

	return &ConfidentialIARP{TracingPoint: T, IssuanceProof: ozkp}
}
