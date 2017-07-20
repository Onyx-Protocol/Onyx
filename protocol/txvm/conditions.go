package txvm

func opDefer(vm *vm) {
	prog := vm.popProgram(datastack)
	vm.pushProgram(entrystack, prog)
}

func opSatisfy(vm *vm) {
	prog := vm.popProgram(entrystack)
	doCommand(vm, prog.program)
}
