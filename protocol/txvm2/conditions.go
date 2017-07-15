package txvm2

func opDefer(vm *vm) {
	prog := vm.popProgram()
	vm.stacks[entrystack].pushTuple(prog)
}

func opSatisfy(vm *vm) {
	prog := vm.stacks[entrystack].popProgram()
	doCommand(vm, prog[1].(vstring))
}
