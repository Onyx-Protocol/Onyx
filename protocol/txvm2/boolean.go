package txvm2

func opNot(vm *vm) {
	b := vm.popBool()
	vm.pushBool(!b)
}

func opAnd(vm *vm) {
	p := vm.popBool()
	q := vm.popBool()
	vm.pushBool(p && q)
}

func opOr(vm *vm) {
	p := vm.popBool()
	q := vm.popBool()
	vm.pushBool(p || q)
}
