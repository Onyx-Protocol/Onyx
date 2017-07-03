package ca

import (
	"chain/crypto/ed25519/ecmath"
)

// AssetIssuanceCandidate interface is used in the following cases:
// * creating IARP: candidates = issuance key specs passed by the user
// * validating IARP: candidates = issuance choices declared in the tx
// This interface helps avoiding reformatting data structures and removes
// code that performs runtime verification such as "is len(assetIDs) == len(issuanceKeys)?"
type AssetIssuanceCandidate interface {
	AssetID() *AssetID
	IssuanceKey() *ecmath.Point
}

type IssuanceAssetRangeProof interface {
	// TODO: add Reader/Writer interfaces
	Validate(
		ac *AssetCommitment,
		candidates []AssetIssuanceCandidate,
		nonce [32]byte,
		message []byte,
	) bool
}

type NonconfidentialIARP struct {
	AssetID AssetID
}

type ConfidentialIARP struct {
	// T is the tracing point
	T           ecmath.Point
	IssuanceZKP *OlegZKP
}

func CreateNonconfidentialIARP(assetID AssetID) *NonconfidentialIARP {
	return &NonconfidentialIARP{AssetID: assetID}
}

func (iarp *NonconfidentialIARP) Validate(ac *AssetCommitment, _ []AssetIssuanceCandidate, _ []byte, _ []byte) bool {
	return ac.Validate(iarp.AssetID, nil)
}

func CreateConfidentialIARP(
	ac *AssetCommitment,
	c ecmath.Scalar,
	candidates []AssetIssuanceCandidate,
	nonce [32]byte,
	msg []byte,
	j uint64, // secret index
	y ecmath.Scalar,
) *ConfidentialIARP {

	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                         {AC, nonce, message,
	//                         uint64le(n),
	//                         a[0],Y[0], ..., a[n-1],Y[n-1]})
	basehash := iarpBasehash(ac, nonce, msg, candidates)

	// 2. Calculate marker point `M`:
	//         M = PointHash("IARP.M", {basehash})
	M := iarpCalcM(basehash)

	// 3. Calculate the tracing point: `T = y·M`.
	var T ecmath.Point
	T.ScMul(&M, &y)

	// 4. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("IARP.msg", {basehash, M, T})
	msghash := hash256("ChainCA.IARP.msg", basehash[:], M.Bytes(), T.Bytes())

	// 5. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("IARP.h", {msghash})
	h := scalarHash("ChainCA.IARP.h", msghash[:])

	F, ok := iarpCommitments(ac, candidates, h, T)
	if !ok {
		panic("Failed to decode an issuance key")
	}

	ozkp := CreateOlegZKP(msghash[:], []ecmath.Scalar{c, y}, iarpFunctions(M, h), F, j)

	return &ConfidentialIARP{T: T, IssuanceZKP: ozkp}
}

func (iarp *ConfidentialIARP) Validate(
	ac *AssetCommitment,
	candidates []AssetIssuanceCandidate,
	nonce [32]byte,
	msg []byte,
) bool {
	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                     {AC, nonce, message,
	//                     uint64le(n),
	//                     a[0],Y[0], ..., a[n-1],Y[n-1]})
	basehash := iarpBasehash(ac, nonce, msg, candidates)

	// 2. Calculate marker point `M`:
	//         M = PointHash("IARP.M", {basehash})
	M := pointHash("ChainCA.IARP.M", basehash[:])

	// 3. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("IARP.msg", {basehash, M, T})
	msghash := hash256("ChainCA.IARP.msg", basehash[:], M.Bytes(), iarp.T.Bytes())

	// 5. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("IARP.h", {msghash})
	h := scalarHash("ChainCA.IARP.h", msghash[:])

	F, ok := iarpCommitments(ac, candidates, h, iarp.T)

	return ok && iarp.IssuanceZKP.Validate(msghash[:], iarpFunctions(M, h), F)
}

func iarpCommitments(
	ac *AssetCommitment,
	candidates []AssetIssuanceCandidate,
	h ecmath.Scalar,
	T ecmath.Point,
) ([][]ecmath.Point, bool) {

	n := len(candidates)

	F := make([][]ecmath.Point, n)
	for i, candidate := range candidates {
		// F[i,0] = AC.H - A[i] + h·Y[i]
		// F[i,1] = AC.C
		// F[i,2] = T
		F[i] = make([]ecmath.Point, 3)

		F[i][0] = *candidate.IssuanceKey()
		a := CreateAssetPoint(candidate.AssetID())
		F[i][0].ScMul(&F[i][0], &h)
		F[i][0].Sub(&F[i][0], (*ecmath.Point)(&a))
		F[i][0].Add(&F[i][0], ac.H())

		F[i][1] = *ac.C()
		F[i][2] = T
	}
	return F, true
}

func iarpFunctions(M ecmath.Point, h ecmath.Scalar) []OlegZKPFunc {
	// f[0](c,y) = (c + h·y)·G
	// f[1](c,y) = c·J
	// f[2](c,y) = y·M
	return []OlegZKPFunc{
		func(scalars []ecmath.Scalar) (P ecmath.Point) {
			// scalars = (c,y)
			var t ecmath.Scalar
			t.Mul(&scalars[1], &h) // y*h
			t.Add(&t, &scalars[0]) // +c
			P.ScMul(&G, &t)        // *G
			return P
		},
		func(scalars []ecmath.Scalar) (P ecmath.Point) {
			// scalars = (c,y)
			P.ScMul(&J, &scalars[0]) // c*J
			return P
		},
		func(scalars []ecmath.Scalar) (P ecmath.Point) {
			// scalars = (c,y)
			P.ScMul(&M, &scalars[1]) // y*M
			return P
		},
	}
}

func iarpBasehash(ac *AssetCommitment, nonce [32]byte, msg []byte, candidates []AssetIssuanceCandidate) [32]byte {
	n := len(candidates)
	hasher := hasher256("ChainCA.IARP.base", ac.Bytes(), nonce[:], msg, uint64le(uint64(n)))
	for _, candidate := range candidates {
		hasher.WriteItem(candidate.AssetID()[:])
		hasher.WriteItem(candidate.IssuanceKey().Bytes())
	}
	var result [32]byte
	hasher.Sum(result[:0])
	return result
}

func iarpCalcM(basehash [32]byte) ecmath.Point {
	return pointHash("ChainCA.IARP.M", basehash[:])
}
