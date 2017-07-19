package txvm2

import (
	"bytes"
	"fmt"
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
