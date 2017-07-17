package txvm2

func opAnnotate(vm *vm) {
	d := vm.popBytes(datastack)
	a := mkAnnotation(d)
	vm.push(effectstack, a)
}
