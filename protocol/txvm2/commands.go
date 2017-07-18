package txvm2

func opCommand(vm *vm) {
	prog := vm.popBytes(datastack)
	doCommand(vm, prog)
}

func doCommand(vm *vm, prog []byte) {
	cmd := mkProgram(prog)
	vm.push(commandstack, cmd)
	defer vm.pop(commandstack)
	exec(vm, prog)
}
