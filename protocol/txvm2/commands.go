package txvm2

func opCommand(vm *vm) {
	prog := vm.popBytes(datastack)
	doCommand(vm, prog)
}

func doCommand(vm *vm, prog []byte) {
	vm.pushProgram(commandstack, &program{prog})
	defer vm.pop(commandstack)
	exec(vm, prog)
}
