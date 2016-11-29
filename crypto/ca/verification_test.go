package ca

import (
	"crypto/rand"
	"testing"
)

var opTrueProgram = []byte{0x51} // OP_TRUE, minus the dependency on protocol/vm

func TestVerifyIssuance(t *testing.T) {
	assetID := AssetID{0xc0, 01}
	amount := uint64(100)

	fi, _, _ := newFakeIssuance(t, assetID, amount)

	err := VerifyIssuance(fi)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifyOutput(t *testing.T) {
	assetID := AssetID{0xc0, 01}
	amount := uint64(100)

	issuance, bf, _ := newFakeIssuance(t, assetID, amount)

	h := []AssetCommitment{issuance.AssetDescriptor().Commitment()}
	cprev := bf.C

	rek := RecordKey{0xbe, 0xef, 0xca, 0xfe}
	ad, vd, arp, vrp, _, _, err := EncryptOutput(rek, assetID, amount, 64, h, cprev, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fo := fakeOutput{ad: ad, vd: vd, arp: arp, vrp: vrp}
	err = VerifyOutput(fo)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifyTxWithExcessCommitment(t *testing.T) {
	assetID := AssetID{0xc0, 01}
	amount := uint64(100)

	issuance, bf, _ := newFakeIssuance(t, assetID, amount)

	h := []AssetCommitment{issuance.AssetDescriptor().Commitment()}
	cprev := bf.C
	rek := RecordKey{0xbe, 0xef, 0xca, 0xfe}
	ad, vd, arp, vrp, c, f, err := EncryptOutput(rek, assetID, amount, 64, h, cprev, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	output := fakeOutput{ad: ad, vd: vd, arp: arp, vrp: vrp}

	// Balance the blinding factors to determine the excess to
	// use in the output.
	q := BalanceBlindingFactors([]BFTuple{bf}, []BFTuple{{C: c, F: f, Value: amount}})
	lc := CreateExcessCommitment(q)

	err = VerifyConfidentialAssets([]Issuance{issuance}, nil, []Output{output}, []ExcessCommitment{lc})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecryptExcessOutput(t *testing.T) {
	assetID := AssetID{0xc0, 01}
	amount := uint64(100)

	issuance, bf, _ := newFakeIssuance(t, assetID, amount)

	h := []AssetCommitment{issuance.AssetDescriptor().Commitment()}
	cprev := bf.C

	// Balance the blinding factors to determine the excess to
	// use in the output.
	excess := BalanceBlindingFactors([]BFTuple{bf}, nil)

	rek := RecordKey{0xbe, 0xef, 0xca, 0xfe}
	ad, vd, _, vrp, _, _, err := EncryptOutput(rek, assetID, amount, 64, h, cprev, nil, &excess)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, err = DecryptOutput(rek, ad, vd, vrp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifySimpleTransferTransaction(t *testing.T) {
	assetID := AssetID{0xc0, 01}
	amount := uint64(100)

	issuance, bf, _ := newFakeIssuance(t, assetID, amount)

	h := []AssetCommitment{issuance.AssetDescriptor().Commitment()}
	cprev := bf.C

	// Balance the blinding factors to determine the excess to
	// use in the output.
	excess := BalanceBlindingFactors([]BFTuple{bf}, nil)

	rek := RecordKey{0xbe, 0xef, 0xca, 0xfe}
	ad, vd, arp, vrp, _, _, err := EncryptOutput(rek, assetID, amount, 64, h, cprev, nil, &excess)
	if err != nil {
		t.Fatal(err)
	}
	output := fakeOutput{ad: ad, vd: vd, arp: arp, vrp: vrp}

	err = VerifyConfidentialAssets([]Issuance{issuance}, nil, []Output{output}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func newFakeIssuance(t *testing.T, assetID AssetID, amount uint64) (Issuance, BFTuple, RecordKey) {
	var rek RecordKey
	_, err := rand.Read(rek[:])
	if err != nil {
		t.Fatal(err)
	}

	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)
	y, Y0 := CreateTransientIssuanceKey(assetID, aek)

	ad, vd, iarp, vrp, c, f, err := EncryptIssuance(rek, assetID, amount, 64, []AssetID{assetID}, []Point{Y0}, y, 1, opTrueProgram)
	if err != nil {
		t.Fatal(err)
	}

	fi := fakeIssuance{
		ad:       ad,
		vd:       vd,
		assetIDs: []AssetID{assetID},
		iarp:     iarp,
		vrp:      vrp,
	}
	return fi, BFTuple{Value: amount, C: c, F: f}, rek
}

type fakeIssuance struct {
	ad       AssetDescriptor
	vd       ValueDescriptor
	assetIDs []AssetID
	iarp     *IssuanceAssetRangeProof
	vrp      *ValueRangeProof
}

func (fi fakeIssuance) AssetDescriptor() AssetDescriptor                  { return fi.ad }
func (fi fakeIssuance) ValueDescriptor() ValueDescriptor                  { return fi.vd }
func (fi fakeIssuance) AssetIDs() []AssetID                               { return fi.assetIDs }
func (fi fakeIssuance) IssuanceAssetRangeProof() *IssuanceAssetRangeProof { return fi.iarp }
func (fi fakeIssuance) ValueRangeProof() *ValueRangeProof                 { return fi.vrp }

type fakeOutput struct {
	ad  AssetDescriptor
	vd  ValueDescriptor
	arp *AssetRangeProof
	vrp *ValueRangeProof
}

func (fo fakeOutput) AssetDescriptor() AssetDescriptor  { return fo.ad }
func (fo fakeOutput) ValueDescriptor() ValueDescriptor  { return fo.vd }
func (fo fakeOutput) AssetRangeProof() *AssetRangeProof { return fo.arp }
func (fo fakeOutput) ValueRangeProof() *ValueRangeProof { return fo.vrp }
