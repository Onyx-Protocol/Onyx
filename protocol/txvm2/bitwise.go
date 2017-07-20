package txvm2

func opBitNot(vm *vm) {
	s := vm.popBytes(datastack)
	for i := 0; i < len(s); i++ {
		s[i] = ^s[i]
	}
	vm.push(datastack, vbytes(s))
}

func opBitAnd(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(vm.errf("bitand: mismatched lengths %d and %d", len(a), len(b)))
	}
	for i := 0; i < len(a); i++ {
		a[i] &= b[i]
	}
	vm.push(datastack, vbytes(a))
}

func opBitOr(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(vm.errf("bitor: mismatched lengths %d and %d", len(a), len(b)))
	}
	for i := 0; i < len(a); i++ {
		a[i] |= b[i]
	}
	vm.push(datastack, vbytes(a))
}

func opBitXor(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	if len(a) != len(b) {
		panic(vm.errf("bitxor: mismatched lengths %d and %d", len(a), len(b)))
	}
	for i := 0; i < len(a); i++ {
		a[i] ^= b[i]
	}
	vm.push(datastack, vbytes(a))
}
