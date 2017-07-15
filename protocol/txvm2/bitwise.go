package txvm2

func opBitNot(vm *vm) {
	s := vm.popString()
	for i := 0; i < len(s); i++ {
		s[i] = ^s[i]
	}
	vm.pushString(s)
}

func opBitAnd(vm *vm) {
	a := vm.popString()
	b := vm.popString()
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] &= b[i]
	}
	vm.pushString(a)
}

func opBitOr(vm *vm) {
	a := vm.popString()
	b := vm.popString()
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] |= b[i]
	}
	vm.pushString(a)
}

func opBitXor(vm *vm) {
	a := vm.popString()
	b := vm.popString()
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] ^= b[i]
	}
	vm.pushString(a)
}
