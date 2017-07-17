package txvm2

func opDefer(vm *vm) {
	prog := vm.popTuple(datastack, programTuple)
	vm.push(entrystack, prog)
}

func opSatisfy(vm *vm) {
	prog := vm.popTuple(entrystack, programTuple)
	doCommand(vm, programProgram(prog))
}
