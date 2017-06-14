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
		inpAssetKeys        []AssetKey
		inpAssetCommitments []*AssetCommitment
		inpAssetBFs         []ecmath.Scalar
		inpValueKeys        []ValueKey
		inpValueCommitments []*ValueCommitment
		inpValueBFs         []ecmath.Scalar
		inpBFTuples         []BFTuple
		outAssetKeys        []AssetKey
		outAssetCommitments []*AssetCommitment
		outAssetBFs         []ecmath.Scalar
		outValueKeys        []ValueKey
		outValueCommitments []*ValueCommitment
		outValueBFs         []ecmath.Scalar
		outBFTuples         []BFTuple
	)

	for i, inp := range inputs {
		aek := []byte{byte(i)}
		inpAssetKeys = append(inpAssetKeys, aek)
		ac, abf := CreateAssetCommitment(inp.assetID, aek)
		inpAssetCommitments = append(inpAssetCommitments, ac)
		inpAssetBFs = append(inpAssetBFs, *abf)

		vek := []byte{byte(i), byte(i)}
		inpValueKeys = append(inpValueKeys, vek)
		vc, vbf := CreateValueCommitment(inp.amount, ac, vek)
		inpValueCommitments = append(inpValueCommitments, vc)
		inpValueBFs = append(inpValueBFs, *vbf)
		inpBFTuples = append(inpBFTuples, BFTuple{inp.amount, *abf, *vbf})
	}
	for i, out := range outputs {
		aek := []byte{byte(10 + i)}
		outAssetKeys = append(outAssetKeys, aek)
		ac, abf := CreateAssetCommitment(out.assetID, aek)
		outAssetCommitments = append(outAssetCommitments, ac)
		outAssetBFs = append(outAssetBFs, *abf)

		vek := []byte{byte(10 + i), byte(10 + i)}
		outValueKeys = append(outValueKeys, vek)
		vc, vbf := CreateValueCommitment(out.amount, ac, vek)
		outValueCommitments = append(outValueCommitments, vc)
		outValueBFs = append(outValueBFs, *vbf)
		outBFTuples = append(outBFTuples, BFTuple{out.amount, *abf, *vbf})
	}

	q := BalanceBlindingFactors(inpBFTuples, outBFTuples)
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
	if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, nil, [][]byte{msg}) {
		t.Error("validated balance of invalid collection of commitments")
	}
	if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, []*ExcessCommitment{qc}, [][]byte{msg[1:]}) {
		t.Error("validated balance of invalid collection of commitments")
	}
}
