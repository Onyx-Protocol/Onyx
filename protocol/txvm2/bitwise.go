package txvm2

func opBitNot(vm *vm) {
	s := vm.popBytes(datastack)
	for i := 0; i < len(s); i++ {
		s[i] = ^s[i]
	}
	vm.push(datastack, s)
}

func opBitAnd(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] &= b[i]
	}
	vm.push(datastack, a)
}

func opBitOr(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] |= b[i]
	}
	vm.push(datastack, a)
}

func opBitXor(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(xxx)
	}
	for i := 0; i < len(a); i++ {
		a[i] ^= b[i]
	}
	vm.push(datastack, a)
}
