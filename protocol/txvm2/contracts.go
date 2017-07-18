package txvm2

func opUnlock(vm *vm) {
	val := vm.popTuple(datastack, valueTuple)
	anchor := vm.popTuple(datastack, anchorTuple)
	cmd := vm.peekTuple(commandstack, programTuple)
	inp := mkContract(val, programProgram(cmd), anchorValue(anchor))
	id := getID(inp)
	vm.push(effectstack, mkInput(id))
	vm.push(entrystack, anchor)
	vm.push(entrystack, val)
}

func opRead(vm *vm) {
	val := vm.popTuple(datastack, valueTuple)
	anchor := vm.popTuple(datastack, anchorTuple)
	cmd := vm.peekTuple(commandstack, programTuple)
	contract := mkContract(val, programProgram(cmd), anchorValue(anchor))
	id := getID(contract)
	vm.push(effectstack, mkRead(id))
}

func opLock(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	anchor := vm.popTuple(entrystack, anchorTuple)
	cmd := vm.peekTuple(commandstack, programTuple)
	contract := mkContract(val, programProgram(cmd), anchorValue(anchor))
	id := getID(contract)
	vm.push(effectstack, mkOutput(id))
}
