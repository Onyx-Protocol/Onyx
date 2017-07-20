package txvm2

func opAnnotate(vm *vm) {
	d := vm.popBytes(datastack)
	vm.pushAnnotation(effectstack, &annotation{d})
}
