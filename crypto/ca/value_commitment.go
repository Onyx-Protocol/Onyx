package ca

import "io"

type ValueCommitment Point

type BFTuple struct {
	Value uint64
	C     Scalar
	F     Scalar
}

func (vc *ValueCommitment) readFrom(r io.Reader) error {
	return (*Point)(vc).readFrom(r)
}

func (vc *ValueCommitment) Bytes() []byte {
	buf := encodePoint((*Point)(vc))
	return buf[:]
}

func (vc *ValueCommitment) FromBytes(b [32]byte) error {
	return (*Point)(vc).fromBytes(&b)
}

func CreateNonblindedValueCommitment(H AssetCommitment, value uint64) ValueCommitment {
	return ValueCommitment(multiplyAndAddPoint(scalarFromUint64(value), Point(H), ZeroScalar))
}

func CreateBlindedValueCommitmentFromBlindingFactor(value uint64, H AssetCommitment, f Scalar) ValueCommitment {
	return ValueCommitment(multiplyAndAddPoint(scalarFromUint64(value), Point(H), f))
}

func CreateBlindedValueCommitment(
	vek ValueKey,
	value uint64,
	H AssetCommitment,
) (ValueCommitment, Scalar) {
	// 1. Calculate `fbuf = SHA3-512(0xbf || vek)`.
	// 2. Calculate `f` as `fbuf` interpreted as a little-endian integer reduced modulo subgroup order `L`: `f = fbuf mod L`.
	f := reducedScalar(hash512([]byte{0xbf}, vek[:]))

	// 3. Calculate point `V = value·H + f·G`.
	V := CreateBlindedValueCommitmentFromBlindingFactor(value, H, f)

	// 4. Return `(V, f)`, where `V` is encoded as a [public key](data.md#public-key) and the blinding factor `f` is encoded as a 256-bit little-endian integer.
	return V, f
}

func BalanceBlindingFactors(inputs, outputs []BFTuple) Scalar {
	// 1. Calculate the sum of input blinding factors: `Finput = ∑(value[j]·c[j]+f[j], j from 0 to n-1) mod L`.
	fInput := ZeroScalar
	for i := 0; i < len(inputs); i++ {
		totalFactor := multiplyAndAddScalars(scalarFromUint64(inputs[i].Value), inputs[i].C, inputs[i].F)
		fInput.Add(&totalFactor)
	}
	// 2. Calculate the sum of output blinding factors: `Foutput = ∑(value’[i]·c’[i]+f’[i], i from 0 to m-1) mod L`.
	fOutput := ZeroScalar
	for i := 0; i < len(outputs); i++ {
		totalFactor := multiplyAndAddScalars(scalarFromUint64(outputs[i].Value), outputs[i].C, outputs[i].F)
		fOutput.Add(&totalFactor)
	}
	// 3. Calculate excess blinding factor as difference between input and output sums: `q = Finput - Foutput mod L`.
	fInput.sub(&fOutput)

	// 4. Return `q`.
	return fInput
}

func VerifyValueCommitmentsBalance(
	inputs []ValueCommitment,
	outputs []ValueCommitment,
	ec []ExcessCommitment,
) bool {
	Pi := ZeroPoint
	for _, inp := range inputs {
		Pi.add((*Point)(&inp))
	}

	Po := ZeroPoint
	for _, out := range outputs {
		Po.add((*Point)(&out))
	}

	for _, l := range ec {
		if !l.Verify() {
			return false
		}
		Po.add(&l.Q)
	}

	return encodePoint(&Pi) == encodePoint(&Po)
}

func (vc *ValueCommitment) String() string {
	return (*Point)(vc).String()
}
