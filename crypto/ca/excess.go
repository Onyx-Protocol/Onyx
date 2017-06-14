package ca

import "chain/crypto/ed25519/ecmath"

type BFTuple struct {
	Amount           uint64
	AssetBF, ValueBF ecmath.Scalar
}

func BalanceBlindingFactors(inputs, outputs []BFTuple) ecmath.Scalar {
	fInput := ecmath.Zero
	for _, inp := range inputs {
		var v ecmath.Scalar
		v.SetUint64(inp.Amount)
		v.MulAdd(&v, &inp.AssetBF, &inp.ValueBF)
		fInput.Add(&fInput, &v)
	}

	fOutput := ecmath.Zero
	for _, out := range outputs {
		var v ecmath.Scalar
		v.SetUint64(out.Amount)
		v.MulAdd(&v, &out.AssetBF, &out.ValueBF)
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

	result.QC.Point1.ScMul(&G, &q)
	result.QC.Point2.ScMul(&J, &q)

	var R1, R2 ecmath.Point
	r := scalarHash("ChainCA.r", result.QC.Point1.Bytes(), result.QC.Point2.Bytes(), q[:], msg)
	R1.ScMul(&G, &r)
	R2.ScMul(&J, &r)

	result.e = scalarHash("ChainCA.EC", result.QC.Point1.Bytes(), result.QC.Point2.Bytes(), R1.Bytes(), R2.Bytes(), msg)
	result.s.MulAdd(&q, &result.e, &r)

	return result
}

func (qc *ExcessCommitment) Validate(msg []byte) bool {
	var R1, R2, T ecmath.Point
	R1.ScMulBase(&qc.s)           // R1 = s·G
	T.ScMul(&qc.QC.Point1, &qc.e) // T = e·QG
	R1.Sub(&R1, &T)               // R1 = s·G - e·QG
	R2.ScMul(&J, &qc.s)           // R2 = s·J
	T.ScMul(&qc.QC.Point2, &qc.e) // T = e·QJ
	R2.Sub(&R2, &T)               // R2 = s·J - e·QJ

	ePrime := scalarHash("ChainCA.EC", qc.QC.Point1.Bytes(), qc.QC.Point2.Bytes(), R1.Bytes(), R2.Bytes(), msg)

	return qc.e == ePrime
}
