package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestValueCommitments(t *testing.T) {
	msg := []byte("message")

	inputs := []struct {
		assetID AssetID
		amount  uint64
	}{
		{AssetID{1}, 10},
		{AssetID{1}, 20},
		{AssetID{2}, 30},
	}
	outputs := []struct {
		assetID AssetID
		amount  uint64
	}{
		{AssetID{1}, 30},
		{AssetID{2}, 5},
		{AssetID{2}, 25},
	}

	var (
		inpAmts             []uint64
		inpAssetBFs         []ecmath.Scalar
		inpValueCommitments []*ValueCommitment
		inpValueBFs         []ecmath.Scalar
		outAmts             []uint64
		outAssetBFs         []ecmath.Scalar
		outValueCommitments []*ValueCommitment
		outValueBFs         []ecmath.Scalar
	)

	for i, inp := range inputs {
		inpAmts = append(inpAmts, inp.amount)

		aek := []byte{byte(i)}
		ac, abf := CreateAssetCommitment(inp.assetID, aek)
		inpAssetBFs = append(inpAssetBFs, *abf)

		vek := []byte{byte(i), byte(i)}
		vc, vbf := CreateValueCommitment(inp.amount, ac, vek)
		inpValueCommitments = append(inpValueCommitments, vc)
		inpValueBFs = append(inpValueBFs, *vbf)
	}
	for i, out := range outputs {
		outAmts = append(outAmts, out.amount)

		aek := []byte{byte(10 + i)}
		ac, abf := CreateAssetCommitment(out.assetID, aek)
		outAssetBFs = append(outAssetBFs, *abf)

		vek := []byte{byte(10 + i), byte(10 + i)}
		vc, vbf := CreateValueCommitment(out.amount, ac, vek)
		outValueCommitments = append(outValueCommitments, vc)
		outValueBFs = append(outValueBFs, *vbf)
	}

	q := BalanceBlindingFactors(inpAmts, outAmts, inpAssetBFs, inpValueBFs, outAssetBFs, outValueBFs)
	qc := CreateExcessCommitment(q, msg)

	if !ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, []*ExcessCommitment{qc}, [][]byte{msg}) {
		t.Error("failed to validate value commitments balance")
	}
	if ValidateValueCommitmentsBalance(inpValueCommitments[1:], outValueCommitments, []*ExcessCommitment{qc}, [][]byte{msg}) {
		t.Error("validated balance of invalid collection of commitments")
	}
	if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments[1:], []*ExcessCommitment{qc}, [][]byte{msg}) {
		t.Error("validated balance of invalid collection of commitments")
	}
	if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, []*ExcessCommitment{qc}, [][]byte{msg[1:]}) {
		t.Error("validated balance of invalid collection of commitments")
	}
}
