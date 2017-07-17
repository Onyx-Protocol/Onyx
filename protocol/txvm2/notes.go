package txvm2

// "notes" are what i'd prefer to call "contracts"

func opUnlock(vm *vm) {
	val := vm.popTuple(datastack, valueTuple)
	anchor := vm.popTuple(datastack, anchorTuple)
	cmd := vm.peekTuple(commandstack, commandTuple)
	inp := mkNote(cmd[1], anchorValue(anchor), val)
	id := getID(inp)
	vm.push(effectstack, mkInput(id))
	vm.push(entrystack, anchor)
	vm.push(entrystack, val)
}

func opRead(vm *vm) {
	val := vm.popTuple(datastack, valueTuple)
	anchor := vm.popTuple(datastack, anchorTuple)
	cmd := vm.peekTuple(commandstack, commandTuple)
	note := mkNote(commandProgram(cmd), anchorValue(anchor), val)
	id := getID(note)
	vm.push(effectstack, mkRead(id))
}

func opLock(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	anchor := vm.popTuple(entrystack, anchorTuple)
	cmd := vm.peekTuple(commandstack, commandTuple)
	note := mkNote(commandProgram(cmd), anchorValue(anchor), val)
	id := getID(note)
	vm.push(effectstack, mkOutput(id))
}
