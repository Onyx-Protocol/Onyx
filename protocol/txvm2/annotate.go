package txvm2

func opAnnotate(vm *vm) {
	d := vm.popString()
	a := mkAnnotation(d)
	vm.stacks[effectstack].pushTuple(a)
}
