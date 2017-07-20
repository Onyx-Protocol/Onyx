package txvm

func opNot(vm *vm) {
	b := vm.popBool(datastack)
	vm.pushBool(datastack, !b)
}

func opAnd(vm *vm) {
	p := vm.popBool(datastack)
	q := vm.popBool(datastack)
	vm.pushBool(datastack, p && q)
}

func opOr(vm *vm) {
	p := vm.popBool(datastack)
	q := vm.popBool(datastack)
	vm.pushBool(datastack, p || q)
}
