package txvm2

// "notes" are what i'd prefer to call "contracts"

func opUnlock(vm *vm) {
	val := vm.popValue()
	anchor := vm.popAnchor()
	cmd := vm.stacks[commandstack].peekCommand()
	inp := mkNote(cmd[1], anchorValue(anchor), val)
	id := getID(inp)
	vm.stacks[effectstack].push(mkInput(id))
	vm.stacks[entrystack].pushTuple(anchor)
	vm.stacks[entrystack].pushTuple(val)
}

func opRead(vm *vm) {
	val := vm.popValue()
	anchor := vm.popAnchor()
	cmd := vm.stacks[commandstack].peekCommand()
	note := mkNote(commandProgram(cmd), anchorValue(anchor), val)
	id := getID(note)
	vm.stacks[effectstack].pushTuple(mkRead(id))
}

func opLock(vm *vm) {
	val := vm.stacks[entrystack].popValue()
	anchor := vm.stacks[entrystack].popAnchor()
	cmd := vm.stacks[commandstack].peekCommand()
	note := mkNote(commandProgram(cmd), anchorValue(anchor), val)
	id := getID(note)
	vm.stacks[effectstack].pushTuple(mkOutput(id))
}
