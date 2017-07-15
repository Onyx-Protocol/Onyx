package txvm2

import "bytes"

func opIssue(vm *vm) {
	amt := vm.popInt64()
	cmd := vm.stacks[commandstack].peekCommand()
	assetDef := mkAssetDefinition(commandProgram(cmd))
	assetID := getID(assetDef)
	vm.stacks[entrystack].pushTuple(mkValue(amt, assetID))
}

func opMerge(vm *vm) {
	v1 := vm.stacks[entrystack].popValue()
	v2 := vm.stacks[entrystack].popValue()
	if !bytes.Equal(valueAssetID(v1), valueAssetID(v2)) {
		panic(xxx)
	}
	vm.stacks[entrystack].pushTuple(mkValue(valueAmount(v1)+valueAmount(v2), valueAssetID(v1)))
}

func opSplit(vm *vm) {
	val := vm.stacks[entrystack].popValue()
	amt := vm.popInt64()
	if amt >= valueAmount(val) {
		panic(xxx)
	}
	vm.stacks[entrystack].pushTuple(mkValue(valueAmount(val)-amt, valueAssetID(val)))
	vm.stacks[entrystack].pushTuple(mkValue(amt, valueAssetID(val)))
}

func opRetire(vm *vm) {
	val := vm.stacks[entrystack].popValue()
	vm.stacks[effectstack].pushTuple(mkRetirement(val))
}
