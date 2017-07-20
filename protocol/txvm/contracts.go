package txvm

func opUnlock(vm *vm) {
	val := vm.popTuple(datastack, valueType, provenvalueType)
	anchor := vm.popAnchor(datastack)
	cmd := vm.peekProgram(commandstack)
	inp := contract{val, cmd.program, anchor.value}
	id := inp.id()
	vm.pushInput(effectstack, &input{id})
	vm.pushAnchor(entrystack, anchor)
	vm.push(entrystack, val.entuple())
}

func opRead(vm *vm) {
	val := vm.popTuple(datastack, valueType, provenvalueType)
	anchor := vm.popAnchor(datastack)
	cmd := vm.peekProgram(commandstack)
	con := contract{val, cmd.program, anchor.value}
	id := con.id()
	vm.pushRead(effectstack, &read{id})
}

func opLock(vm *vm) {
	val := vm.popTuple(entrystack, valueType, provenvalueType)
	anchor := vm.popAnchor(entrystack)
	cmd := vm.peekProgram(commandstack)
	con := contract{val, cmd.program, anchor.value}
	id := con.id()
	vm.pushOutput(effectstack, &output{id})
}
