package txvm2

import (
	"chain/crypto/ca"
	"chain/crypto/ed25519/ecmath"
)

func opWrapValue(vm *vm) {
	// xxx
}

func opMergeConfidential(vm *vm) {
	a := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)
	b := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)

	getValueCommitment := func(a tuple) *ca.ValueCommitment {
		name, _ := a.name()
		if name == valueTuple {
			var assetID ca.AssetID
			copy(assetID[:], valueAssetID(a))
			ac, _ := ca.CreateAssetCommitment(assetID, nil)
			vc, _ := ca.CreateValueCommitment(uint64(valueAmount(a)), ac, nil)
			return vc
		}
		var vctuple tuple
		if name == provenValueTuple {
			vctuple = provenValueValueCommitment(a)
		} else {
			vctuple = unprovenValueValueCommitment(a)
		}
		var V, F ecmath.Point
		var pointBytes [32]byte
		copy(pointBytes[:], valueCommitmentValuePoint(vctuple))
		_, ok := V.Decode(pointBytes)
		if !ok {
			panic("mergeconfidential: invalid curve point")
		}
		copy(pointBytes[:], valueCommitmentBlindingPoint(vctuple))
		_, ok = F.Decode(pointBytes)
		if !ok {
			panic("mergeconfidential: invalid curve point")
		}
		return &ca.ValueCommitment{V, F}
	}

	vca := getValueCommitment(a)
	vcb := getValueCommitment(b)
	vca.Add(vca, vcb)
	vcbytes := vca.Bytes()
	vc := mkValueCommitment(vbytes(vcbytes[:32]), vbytes(vcbytes[32:]))
	vm.push(entrystack, mkUnprovenValue(vc))
}

func opSplitConfidential(vm *vm) {
	// xxx
}

func opProveAssetRange(vm *vm) {
	// xxx
}

func opDropAssetCommitment(vm *vm) {
	// xxx
}

func opProveAssetID(vm *vm) {
	// xxx
}

func opProveAmount(vm *vm) {
	// xxx
}

func opProveValueRange(vm *vm) {
	// xxx
}

func opIssuanceCandidate(vm *vm) {
	// xxx
}

func opIssueConfidential(vm *vm) {
	// xxx
}
