package txvm2

import (
	"bytes"

	"chain/math/checked"
)

func opIssue(vm *vm) {
	amt := vm.popInt64(datastack)
	cmd := vm.peekProgram(commandstack)
	assetDef := assetdefinition{cmd.program}
	assetID := assetDef.id()
	vm.pushValue(entrystack, &value{amt, assetID})
}

func opMerge(vm *vm) {
	v1 := vm.popValue(entrystack)
	v2 := vm.popValue(entrystack)
	if !bytes.Equal(v1.assetID, v2.assetID) {
		panic(vm.errf("merge: mismatched asset IDs (%x vs. %x)", v1.assetID, v2.assetID))
	}
	newamt, ok := checked.AddInt64(v1.amount, v2.amount)
	if !ok {
		panic(vm.errf("merge: sum of %d and %d overflows int64", v1.amount, v2.amount))
	}
	vm.pushValue(entrystack, &value{newamt, v1.assetID})
}

func opSplit(vm *vm) {
	val := vm.popValue(entrystack)
	amt := vm.popInt64(datastack)
	if amt >= val.amount {
		panic(vm.errf("split: amount too large (%d vs. %d)", amt, val.amount))
	}
	vm.pushValue(entrystack, &value{val.amount - amt, val.assetID})
	vm.pushValue(entrystack, &value{amt, val.assetID})
}

func opRetire(vm *vm) {
	val := vm.popTuple(entrystack, valueType, provenvalueType)
	_, vc := toCommitments(val)
	vm.pushRetirement(effectstack, &retirement{valuecommitment{vc}})
}
