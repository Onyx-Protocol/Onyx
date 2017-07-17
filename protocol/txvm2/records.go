package txvm2

import "bytes"

func opCreate(vm *vm) {
	data := vm.pop(datastack)
	cmd := vm.peekTuple(commandstack, commandTuple)
	rec := mkRecord(commandProgram(cmd), data)
	vm.push(entrystack, rec)
}

func opDelete(vm *vm) {
	rec := vm.popTuple(recordstack, recordTuple)
	cmd := vm.peekTuple(commandStack, commandTuple)
	if !bytes.Equal(recordCommandProgram(rec), commandProgram(cmd)) {
		panic(xxx)
	}
}

func opComplete(vm *vm) {
	rec := vm.popTuple(recordstack, recordTuple)
	cmd := vm.peekTuple(commandstack, commandTuple)
	if !bytes.Equal(recordCommandProgram(rec), commandProgram(cmd)) {
		panic(xxx)
	}
	vm.push(effectstack, rec)
}
