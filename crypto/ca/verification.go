package ca

import (
	"fmt"

	"chain-stealth/errors"
)

var ErrUnbalanced = errors.New("value commitments do not balance")

// Inputs
// 1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
// 2. `VD`: the [value commitment](#value-descriptor).
// 3. `ARP`: the [asset range proof](#asset-range-proof).
// 4. `VRP`: the [value range proof](#value-range-proof).
func VerifyOutput(output Output) error {
	ad := output.AssetDescriptor()
	arp := output.AssetRangeProof()
	vd := output.ValueDescriptor()
	vrp := output.ValueRangeProof()

	// 1. If `ARP` is not empty and `AD` is blinded:
	if arp != nil && ad.IsBlinded() {
		// 1. [Verify asset range proof](#verify-asset-range-proof) using `(AD.H,AD.(ea,ec),ARP)`. If verification failed, halt and return `false`.
		if err := arp.Verify(ad.Commitment(), ad.EncryptedAssetID()); err != nil {
			return errors.Wrap(err, "asset range proof verification")
		}
	}

	// 2. If `VRP` is not empty and `VD` is blinded:
	if vrp != nil && vd.IsBlinded() {
		// 1. [Verify value range proof](#verify-value-range-proof) using `(AD.H,VD.V,VD.(ev,ef),VRP)`. If verification failed, halt and return `false`.
		if err := vrp.Verify(ad.Commitment(), vd.Commitment(), vd.EncryptedValue()); err != nil {
			return errors.Wrap(err, "value range proof verification")
		}
	}
	// 3. Return `true`.
	return nil
}

// Inputs:
// 1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
// 2. `VD`: the [value descriptor](#value-descriptor).
// 3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).
// 4. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
// 5. `VRP`: the [value range proof](#value-range-proof).
func VerifyIssuance(issuance Issuance) error {

	ad := issuance.AssetDescriptor()
	vd := issuance.ValueDescriptor()
	assetids := issuance.AssetIDs()
	iarp := issuance.IssuanceAssetRangeProof()
	vrp := issuance.ValueRangeProof()

	// 1. If `IARP` is not empty and `AD` is blinded:
	if iarp != nil && ad.IsBlinded() {
		// 1. [Verify issuance asset range proof](#verify-issuance-asset-range-proof) using `(IARP,AD.H,{a[i]})`. If verification failed, halt and return `false`.
		if err := iarp.Verify(ad.Commitment(), assetids); err != nil {
			return errors.Wrap(err, "issuance asset range proof verification")
		}
	}
	// 2. If `VRP` is not empty and `VD` is blinded:
	if vrp != nil && vd.IsBlinded() {
		// 1. [Verify value range proof](#verify-value-range-proof) using `(AD.H, VD.V, evef=(0x00...,0x00...),VRP)`. If verification failed, halt and return `false`.
		if err := vrp.Verify(ad.Commitment(), vd.Commitment(), vd.EncryptedValue()); err != nil {
			return errors.Wrap(err, "value range proof verification")
		}
	}
	// 3. Return `true`.
	return nil
}

// Inputs:
// 1. List of issuances, each input consisting of:
//     * `AD`: the [asset ID descriptor](#asset-id-descriptor).
//     * `VD`: the [value descriptor](#value-descriptor).
//     * `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).
//     * `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
//     * `VRP`: the [value range proof](#value-range-proof).
// 2. List of inputs, each input consisting of:
//     * `AD`: the [asset ID descriptor](#asset-id-descriptor).
//     * `VD`: the [value descriptor](#value-descriptor).
// 3. List of outputs, each output consisting of:
//     * `AD`: the [asset ID descriptor](#asset-id-descriptor).
//     * `VD`: the [value descriptor](#value-descriptor).
//     * `ARP`: the [asset range proof](#asset-range-proof) or empty string.
//     * `VRP`: the [value range proof](#value-range-proof) or empty string.
// 4. The list of [excess commitments](#excess-commitment): `{(Q[i], s[i], e[i])}`.
func VerifyConfidentialAssets(
	issuances []Issuance,
	spends []Spend,
	outputs []Output,
	excessCommits []ExcessCommitment,
) error {
	// 1. [Verify each issuance](#verify-issuance). If verification failed, halt and return `false`.
	for i, issuance := range issuances {
		err := VerifyIssuance(issuance)
		if err != nil {
			return errors.Wrapf(err, "verification of issuance %d", i)
		}
	}
	// 2. For each output:
	for i, output := range outputs {
		ad := output.AssetDescriptor()
		if ad.IsBlinded() {
			// 1. If `AD` is blinded and `ARP` is an empty string, or an ARP with zero keys, verify that `AD.H` equals one of the asset ID commitments in the spends or issuances. If not, halt and return `false`.
			arp := output.AssetRangeProof()
			if arp == nil || len(arp.H) == 0 {
				ac := ad.Commitment()
				if !checkAssetCommitmentPresence(ac, issuances, spends) {
					return fmt.Errorf("output %d: asset commitment not present in inputs", i)
				}

				// 2. If `AD` is blinded and `ARP` is not empty, verify that each asset ID commitment in the `ARP` belongs to the set of the asset ID commitments on the spends and issuances. If not, halt and return `false`.
			} else {
				for _, H := range arp.H {
					if !checkAssetCommitmentPresence(H, issuances, spends) {
						return fmt.Errorf("output %d: asset commitment not present in inputs", i)
					}
				}
			}
		}
		// 3. If there are more than one output and the output’s value descriptor is blinded:
		if len(outputs) > 1 && output.ValueDescriptor().IsBlinded() {
			// 1. Verify that the value range proof is not empty. Otherwise, halt and return `false`.
			if output.ValueRangeProof() == nil {
				return fmt.Errorf("output %d: missing value range proof", i)
			}
		}

		// 4. [Verify output](#verify-output). If verification failed, halt and return `false`.
		err := VerifyOutput(output)
		if err != nil {
			return errors.Wrapf(err, "verifying output %d", i)
		}
	}
	// 3. [Verify value commitments balance](#verify-value-commitments-balance) using a union of issuance and input value commitments as input commitments. If verification failed, halt and return `false`.
	ins := make([]ValueCommitment, len(issuances)+len(spends))
	outs := make([]ValueCommitment, len(outputs))
	for i, issuance := range issuances {
		ins[i] = issuance.ValueDescriptor().Commitment()
	}
	for i, spend := range spends {
		ins[i+len(issuances)] = spend.ValueDescriptor().Commitment()
	}
	for i, output := range outputs {
		outs[i] = output.ValueDescriptor().Commitment()
	}
	if !VerifyValueCommitmentsBalance(ins, outs, excessCommits) {
		return ErrUnbalanced
	}

	// 4. Return `true`.
	return nil
}

func checkAssetCommitmentPresence(H AssetCommitment, issuances []Issuance, spends []Spend) bool {
	// Find each of Hs in issuances or spends.
	for _, issuance := range issuances {
		if issuance.AssetDescriptor().Commitment() == H {
			return true
		}
	}
	for _, spend := range spends {
		if spend.AssetDescriptor().Commitment() == H {
			return true
		}
	}
	// H not found among issuances and spends
	return false
}
