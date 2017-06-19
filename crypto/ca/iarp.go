package ca

import (
	"chain/crypto/ed25519/ecmath"
)

type IssuanceAssetRangeProof interface {
	// TODO: add Reader/Writer interfaces
	Validate(
		ac *AssetCommitment,
		assetIDs []AssetID,
		Y []ecmath.Point,
		nonce []byte,
		message []byte,
	) bool
}

type NonconfidentialIARP struct {
	AssetID AssetID
}

type ConfidentialIARP struct {
	// Y is the list of issuance keys
	Y []ecmath.Point

	// T is the tracing point
	T           ecmath.Point
	IssuanceZKP *OlegZKP
}

func CreateNonconfidentialIARP(assetID AssetID) *NonconfidentialIARP {
	return &NonconfidentialIARP{AssetID: assetID}
}

func (iarp *NonconfidentialIARP) Validate(ac *AssetCommitment, _ []AssetID, _ []ecmath.Point, _ []byte, _ []byte) bool {
	return ac.Validate(iarp.AssetID, nil)
}

func CreateConfidentialIARP(
	ac *AssetCommitment,
	c ecmath.Scalar,
	assetIDs []AssetID,
	Y []ecmath.Point, // issuance keys
	nonce [32]byte,
	msg []byte,
	secretIndex uint64,
	y ecmath.Scalar,
) *ConfidentialIARP {

	n := len(assetIDs)
	if len(Y) != n {
		panic("calling error")
	}

	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                         {AC, nonce, message,
	//                         uint64le(n),
	//                         a[0],Y[0], ..., a[n-1],Y[n-1]})
	basehash := iarpBasehash(ac, nonce, msg, assetIDs, Y)

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

	F, ok := iarpCommitments(ac, assetIDs, Y, h, T)
	if !ok {
		panic("Failed to decode an issuance key")
	}

	ozkp := CreateOlegZKP(
		msghash[:],
		[]ecmath.Scalar{c, y},
		iarpFunctions(M, h),
		F,
		secretIndex,
	)

	return &ConfidentialIARP{Y: Y, T: T, IssuanceZKP: ozkp}
}

func (iarp *ConfidentialIARP) Validate(
	ac *AssetCommitment,
	assetIDs []AssetID,
	Y []ecmath.Point,
	nonce [32]byte,
	msg []byte,
) bool {
	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                     {AC, nonce, message,
	//                     uint64le(n),
	//                     a[0],Y[0], ..., a[n-1],Y[n-1]})
	basehash := iarpBasehash(ac, nonce, msg, assetIDs, Y)

	// 2. Calculate marker point `M`:
	//         M = PointHash("IARP.M", {basehash})
	M := pointHash("ChainCA.IARP.M", basehash[:])

	// 3. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("IARP.msg", {basehash, M, T})
	msghash := hash256("ChainCA.IARP.msg", basehash[:], M.Bytes(), iarp.T.Bytes())

	// 5. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("IARP.h", {msghash})
	h := scalarHash("ChainCA.IARP.h", msghash[:])

	F, ok := iarpCommitments(ac, assetIDs, Y, h, iarp.T)

	if !ok {
		return false
	}

	return iarp.IssuanceZKP.Validate(
		msghash[:],
		iarpFunctions(M, h),
		F)

	return true
}

func iarpCommitments(
	ac *AssetCommitment,
	assetIDs []AssetID,
	Y []ecmath.Point,
	h ecmath.Scalar,
	T ecmath.Point,
) ([][]ecmath.Point, bool) {

	n := len(assetIDs)
	if len(Y) != n {
		panic("calling error")
	}

	F := make([][]ecmath.Point, n)
	for i, assetID := range assetIDs {
		// F[i,0] = AC.H - A[i] + h·Y[i]
		// F[i,1] = AC.C
		// F[i,2] = T
		F[i] = make([]ecmath.Point, 3)

		F[i][0] = Y[i]
		a := CreateAssetPoint(&assetID)
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

func iarpBasehash(ac *AssetCommitment, nonce [32]byte, msg []byte, assetIDs []AssetID, Y []ecmath.Point) [32]byte {
	n := len(assetIDs)
	if len(Y) != n {
		panic("calling error")
	}
	hasher := hasher256("ChainCA.IARP.base", ac.Bytes(), nonce[:], msg, uint64le(uint64(n)))
	for i, assetID := range assetIDs {
		hasher.WriteItem(assetID[:])
		hasher.WriteItem(Y[i].Bytes())
	}
	var result [32]byte
	hasher.Sum(result[:0])
	return result
}

func iarpCalcM(basehash [32]byte) ecmath.Point {
	return pointHash("ChainCA.IARP.M", basehash[:])
}
