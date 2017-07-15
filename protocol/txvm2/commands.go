package txvm2

func opCommand(vm *vm) {
	prog := vm.popString()
	doCommand(vm, prog)
}

func doCommand(vm *vm, prog []byte) {
	cmd := mkCommand(prog)
	vm.stacks[commandstack].pushTuple(cmd)
	defer vm.stacks[commandstack].pop()
	exec(vm, prog)
}
