package ca

import "chain/crypto/ed25519/ecmath"

func BalanceBlindingFactors(inAmts, outAmts []uint64, inAssetBFs, inValueBFs, outAssetBFs, outValueBFs []ecmath.Scalar) ecmath.Scalar {
	n := len(inAmts)
	if len(inAssetBFs) != n {
		panic("calling error")
	}
	if len(inValueBFs) != n {
		panic("calling error")
	}
	m := len(outAmts)
	if len(outAssetBFs) != m {
		panic("calling error")
	}
	if len(outValueBFs) != m {
		panic("calling error")
	}

	fInput := ecmath.Zero
	for i, amt := range inAmts {
		var (
			assetBF = inAssetBFs[i]
			valueBF = inValueBFs[i]
			v       ecmath.Scalar
		)
		v.SetUint64(amt)
		v.MulAdd(&v, &assetBF, &valueBF)
		fInput.Add(&fInput, &v)
	}

	fOutput := ecmath.Zero
	for i, amt := range outAmts {
		var (
			assetBF = outAssetBFs[i]
			valueBF = outValueBFs[i]
			v       ecmath.Scalar
		)
		v.SetUint64(amt)
		v.MulAdd(&v, &assetBF, &valueBF)
		fOutput.Add(&fOutput, &v)
	}

	var q ecmath.Scalar
	q.Sub(&fInput, &fOutput)

	return q
}

type ExcessCommitment struct {
	QC   PointPair
	e, s ecmath.Scalar
}

func CreateExcessCommitment(q ecmath.Scalar, msg []byte) *ExcessCommitment {
	result := new(ExcessCommitment)

	result.QC[0].ScMul(&G, &q)
	result.QC[1].ScMul(&J, &q)

	var R1, R2 ecmath.Point
	r := scalarHash("ChainCA.r", result.QC[0].Bytes(), result.QC[1].Bytes(), q[:], msg)
	R1.ScMul(&G, &r)
	R2.ScMul(&J, &r)

	result.e = scalarHash("ChainCA.EC", result.QC[0].Bytes(), result.QC[1].Bytes(), R1.Bytes(), R2.Bytes(), msg)
	result.s.MulAdd(&q, &result.e, &r)

	return result
}

func (qc *ExcessCommitment) Validate(msg []byte) bool {
	var R1, R2, T ecmath.Point
	R1.ScMulBase(&qc.s)       // R1 = s·G
	T.ScMul(&qc.QC[0], &qc.e) // T = e·QG
	R1.Sub(&R1, &T)           // R1 = s·G - e·QG
	R2.ScMul(&J, &qc.s)       // R2 = s·J
	T.ScMul(&qc.QC[1], &qc.e) // T = e·QJ
	R2.Sub(&R2, &T)           // R2 = s·J - e·QJ

	ePrime := scalarHash("ChainCA.EC", qc.QC[0].Bytes(), qc.QC[1].Bytes(), R1.Bytes(), R2.Bytes(), msg)

	return qc.e == ePrime
}
