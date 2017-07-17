package txvm2

import "bytes"

func opIssue(vm *vm) {
	amt := vm.popInt64(datastack)
	cmd := vm.peekTuple(commandstack, commandTuple)
	assetDef := mkAssetDefinition(commandProgram(cmd))
	assetID := getID(assetDef)
	vm.push(entrystack, mkValue(amt, assetID))
}

func opMerge(vm *vm) {
	v1 := vm.popTuple(entrystack, valueTuple)
	v2 := vm.popTuple(entrystack, valueTuple)
	if !bytes.Equal(valueAssetID(v1), valueAssetID(v2)) {
		panic(xxx)
	}
	vm.push(entrystack, mkValue(valueAmount(v1)+valueAmount(v2), valueAssetID(v1)))
}

func opSplit(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	amt := vm.popInt64(datastack)
	if amt >= valueAmount(val) {
		panic(xxx)
	}
	vm.push(entrystack, mkValue(valueAmount(val)-amt, valueAssetID(val)))
	vm.push(entrystack, mkValue(amt, valueAssetID(val)))
}

func opRetire(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	vm.push(effectstack, mkRetirement(val))
}

func opMergeConfidential(vm *vm) {
	// xxx
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
