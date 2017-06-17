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
	Validate(
		ac *AssetCommitment,
		issuanceKeyTuples []AssetIssuanceKeyTuple,
		nonce []byte,
		message []byte,
	) bool
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

func (iarp *NonconfidentialIARP) Validate(
	ac *AssetCommitment,
	issuanceKeyTuples []AssetIssuanceKeyTuple,
	nonce []byte,
	message []byte,
) bool {
	ac2 := PointPair{ecmath.Point(CreateAssetPoint(&iarp.AssetID)), ecmath.ZeroPoint}
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
	h := scalarHash("ChainCA.IARP.h", msghash[:])

	F, ok := iarpCommitments(ac, issuanceKeyTuples, &h, &T)
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

	return &ConfidentialIARP{TracingPoint: T, IssuanceProof: ozkp}
}

func (iarp *ConfidentialIARP) Validate(
	ac *AssetCommitment,
	issuanceKeyTuples []AssetIssuanceKeyTuple,
	nonce []byte,
	message []byte,
) bool {
	n := uint64(len(issuanceKeyTuples))

	// 1. Calculate the base hash:
	//         basehash = Hash256("IARP.base",
	//                     {AC, nonce, message,
	//                     uint64le(n),
	//                     a[0],Y[0], ..., a[n-1],Y[n-1]})
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

	// 3. Calculate a 32-byte message hash to sign:
	//         msghash = Hash256("IARP.msg", {basehash, M, T})
	msghash := hash256("ChainCA.IARP.msg", basehash[:], M.Bytes(), iarp.TracingPoint.Bytes())

	// 5. Calculate Fiat-Shamir challenge `h` for the issuance key:
	//         h = ScalarHash("IARP.h", {msghash})
	h := scalarHash("ChainCA.IARP.h", msghash[:])

	F, ok := iarpCommitments(ac, issuanceKeyTuples, &h, &iarp.TracingPoint)

	if !ok {
		return false
	}

	return iarp.IssuanceProof.Validate(
		msghash[:],
		iarpFunctions(M, h),
		F)

	return true
}

func iarpCommitments(
	ac *AssetCommitment,
	issuanceKeyTuples []AssetIssuanceKeyTuple,
	h *ecmath.Scalar,
	T *ecmath.Point,
) ([][]ecmath.Point, bool) {

	n := len(issuanceKeyTuples)
	F := make([][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		// F[i,0] = AC.H - A[i] + h·Y[i]
		// F[i,1] = AC.C
		// F[i,2] = T
		F[i] = make([]ecmath.Point, 3)

		y := issuanceKeyTuples[i].IssuanceKey()
		F[i][0] = ecmath.ZeroPoint
		if F[i][0].UnmarshalBinary(y) != nil {
			return nil, false
		}
		a := CreateAssetPoint(issuanceKeyTuples[i].AssetID())
		F[i][0].ScMul(&F[i][0], h)
		F[i][0].Sub(&F[i][0], (*ecmath.Point)(&a))
		F[i][0].Add(&F[i][0], ac.H())

		F[i][1] = *ac.C()
		F[i][2] = *T
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
