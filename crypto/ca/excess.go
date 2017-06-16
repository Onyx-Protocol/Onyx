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

const ExcessCommitmentMinSize = 128

type ExcessCommitment struct {
	QC   PointPair
	e, s ecmath.Scalar
	msg  []byte
}

// Decode attempts to decode excess commitment.
// Returns false if buffer is smaller than ExcessCommitmentMinSize or
// first 64 bytes are not a valid pair of Ed25519 points.
func (ec *ExcessCommitment) Decode(buf []byte) bool {
	if len(buf) < ExcessCommitmentMinSize {
		return false
	}
	var pairbuf [64]byte
	copy(pairbuf[:], buf[0:64])
	_, ok := ec.QC.Decode(pairbuf)
	if !ok {
		return false
	}
	copy(ec.e[:], buf[64:96])
	copy(ec.s[:], buf[96:128])
	ec.msg = make([]byte, len(buf)-ExcessCommitmentMinSize)
	copy(ec.msg, buf[128:])
	return true
}

// Bytes returns a binary encoding of the excess commitment.
func (ec *ExcessCommitment) Bytes() []byte {
	result := make([]byte, ExcessCommitmentMinSize+len(ec.msg))
	copy(result[0:64], ec.QC.Bytes())
	copy(result[64:96], ec.e[:])
	copy(result[96:128], ec.s[:])
	copy(result[128:], ec.msg)
	return result
}

// SignatureBytes returns a binary encoding
// of the Schnorr signature part {e,s} (64 bytes).
func (ec *ExcessCommitment) SignatureBytes() []byte {
	result := make([]byte, 64)
	copy(result[0:32], ec.e[:])
	copy(result[32:64], ec.s[:])
	return result
}

// CreateExcessCommitment returns a valid excess commitment
// for a given excess factor q with a message msg.
func CreateExcessCommitment(q ecmath.Scalar, msg []byte) *ExcessCommitment {
	result := new(ExcessCommitment)

	result.msg = make([]byte, len(msg))
	copy(result.msg, msg)

	result.QC[0].ScMul(&G, &q)
	result.QC[1].ScMul(&J, &q)

	var R1, R2 ecmath.Point
	r := scalarHash("ChainCA.EC.r", result.QC[0].Bytes(), result.QC[1].Bytes(), q[:], msg)
	R1.ScMul(&G, &r)
	R2.ScMul(&J, &r)

	// e = ScalarHash("EC", {QG, QJ, R1, R2, msg})
	result.e = scalarHash("ChainCA.EC", result.QC[0].Bytes(), result.QC[1].Bytes(), R1.Bytes(), R2.Bytes(), msg)
	result.s.MulAdd(&q, &result.e, &r)

	return result
}

// Validate returns true if the signature within the excess commitment is valid.
func (ec *ExcessCommitment) Validate() bool {
	var R1, R2, T ecmath.Point
	R1.ScMulBase(&ec.s)       // R1 = s·G
	T.ScMul(&ec.QC[0], &ec.e) // T = e·QG
	R1.Sub(&R1, &T)           // R1 = s·G - e·QG
	R2.ScMul(&J, &ec.s)       // R2 = s·J
	T.ScMul(&ec.QC[1], &ec.e) // T = e·QJ
	R2.Sub(&R2, &T)           // R2 = s·J - e·QJ

	e := scalarHash("ChainCA.EC", ec.QC[0].Bytes(), ec.QC[1].Bytes(), R1.Bytes(), R2.Bytes(), ec.msg)

	return ec.e == e
}
