package ca

import "testing"

func (vc *ValueCommitment) flipBits(f func()) {
	(*Point)(vc).flipBits(f)
}

func TestNonblindedValueCommitment(t *testing.T) {
	want := ValueCommitment(mustDecodePoint(fromHex256("821a9df04184630ddec3b1bab1f862d2c0a46389780368c09dcb3699dfb5b0c5")))
	got := CreateNonblindedValueCommitment(AssetCommitment(mustDecodePoint(fromHex256("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f"))), 123)
	if got != want {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func TestBlindedValueCommitment(t *testing.T) {
	var rek RecordKey
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)
	vek := DeriveValueKey(iek)

	var assetID AssetID

	H := CreateNonblindedAssetCommitment(assetID)

	H1, c1 := CreateBlindedAssetCommitment(H, ZeroScalar, aek)
	H2, c2 := CreateBlindedAssetCommitment(H1, c1, aek)

	Vi1, fi1 := CreateBlindedValueCommitment(vek, 1, H1)
	Vi2, fi2 := CreateBlindedValueCommitment(vek, 2, H1)
	Vi3, fi3 := CreateBlindedValueCommitment(vek, 4, H2)
	Vi4, fi4 := CreateBlindedValueCommitment(vek, 8, H2)

	Vo1, fo1 := CreateBlindedValueCommitment(vek, 3, H1)
	Vo2, fo2 := CreateBlindedValueCommitment(vek, 12, H2)

	inputBFTuples := []BFTuple{
		{
			Value: 1,
			C:     c1,
			F:     fi1,
		},
		{
			Value: 2,
			C:     c1,
			F:     fi2,
		},
		{
			Value: 4,
			C:     c2,
			F:     fi3,
		},
		{
			Value: 8,
			C:     c2,
			F:     fi4,
		},
	}
	outputBFTuples := []BFTuple{
		{
			Value: 3,
			C:     c1,
			F:     fo1,
		},
		{
			Value: 12,
			C:     c2,
			F:     fo2,
		},
	}

	q := BalanceBlindingFactors(inputBFTuples, outputBFTuples)
	lc := CreateExcessCommitment(q)

	inputVCs := []ValueCommitment{Vi1, Vi2, Vi3, Vi4}
	outputVCs := []ValueCommitment{Vo1, Vo2}

	if !VerifyValueCommitmentsBalance(inputVCs, outputVCs, []ExcessCommitment{lc}) {
		t.Error("input/output (and excess) commitments do not balance")
	}

	test := func() {
		if VerifyValueCommitmentsBalance(inputVCs, outputVCs, []ExcessCommitment{lc}) {
			t.Error("unexpected success verifying value commitments balance")
		}
	}
	for i := range inputVCs {
		inputVCs[i].flipBits(test)
	}
	for i := range outputVCs {
		outputVCs[i].flipBits(test)
	}
}
