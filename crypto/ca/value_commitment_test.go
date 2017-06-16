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

	f := func(balanced bool) func(*testing.T) {
		return func(t *testing.T) {
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
				amt := inp.amount
				if i == 0 && !balanced {
					amt++
				}

				inpAmts = append(inpAmts, amt)

				aek := []byte{byte(i)}
				ac, abf := CreateAssetCommitment(inp.assetID, aek)
				inpAssetBFs = append(inpAssetBFs, *abf)

				vek := []byte{byte(i), byte(i)}
				vc, vbf := CreateValueCommitment(amt, ac, vek)
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

			res := ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, []*ExcessCommitment{qc})

			if balanced {
				if !res {
					t.Error("failed to validate value commitments balance")
				}
			} else {
				if res {
					t.Error("validated unbalanced value commitments")
				}
			}
			if ValidateValueCommitmentsBalance(inpValueCommitments[1:], outValueCommitments, []*ExcessCommitment{qc}) {
				t.Error("validated balance of invalid collection of commitments")
			}
			if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments[1:], []*ExcessCommitment{qc}) {
				t.Error("validated balance of invalid collection of commitments")
			}
			qc.msg = qc.msg[1:]
			if ValidateValueCommitmentsBalance(inpValueCommitments, outValueCommitments, []*ExcessCommitment{qc}) {
				t.Error("validated balance of invalid collection of commitments")
			}
		}
	}

	t.Run("balanced", f(true))
	t.Run("unbalanced", f(false))
}
