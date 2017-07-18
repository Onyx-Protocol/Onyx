package txvm2

import (
	"bytes"
	"fmt"

	"chain/crypto/ca"
	"chain/crypto/ed25519/ecmath"
)

func opIssue(vm *vm) {
	amt := vm.popInt64(datastack)
	cmd := vm.peekTuple(commandstack, programTuple)
	assetDef := mkAssetDefinition(programProgram(cmd))
	assetID := getID(assetDef)
	vm.push(entrystack, mkValue(amt, assetID))
}

func opMerge(vm *vm) {
	v1 := vm.popTuple(entrystack, valueTuple)
	v2 := vm.popTuple(entrystack, valueTuple)
	if !bytes.Equal(valueAssetID(v1), valueAssetID(v2)) {
		panic(fmt.Errorf("merge: mismatched asset IDs (%x vs. %x)", valueAssetID(v1), valueAssetID(v2)))
	}
	vm.push(entrystack, mkValue(valueAmount(v1)+valueAmount(v2), valueAssetID(v1)))
}

func opSplit(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	amt := vm.popInt64(datastack)
	if amt >= valueAmount(val) {
		panic(fmt.Errorf("split: amount too large (%d vs. %d)", amt, valueAmount(val)))
	}
	vm.push(entrystack, mkValue(valueAmount(val)-amt, valueAssetID(val)))
	vm.push(entrystack, mkValue(amt, valueAssetID(val)))
}

func opRetire(vm *vm) {
	// xxx needs to be in terms of value commitments
	val := vm.popTuple(entrystack, valueTuple)
	vm.push(effectstack, mkRetirement(val))
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
