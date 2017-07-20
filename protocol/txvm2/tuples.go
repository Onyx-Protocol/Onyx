package txvm2

func opTuple(vm *vm) {
	n := vm.popInt64(datastack)
	var vals []Item
	for n > 0 {
		v := vm.pop(datastack)
		vals = append(vals, v)
		n--
	}
	vm.push(datastack, tuple(vals))
}

func opUntuple(vm *vm) {
	v := vm.pop(datastack)
	t, ok := v.(tuple)
	if !ok {
		panic(vm.errf("untuple: %T is not a tuple", v))
	}
	for i := len(t) - 1; i >= 0; i-- {
		vm.push(datastack, t[i])
	}
	vm.push(datastack, vint64(len(t)))
}

func opField(vm *vm) {
	n := vm.popInt64(datastack)
	v := vm.pop(datastack)
	t, ok := v.(tuple)
	if !ok {
		panic(vm.errf("field: %T is not a tuple", v))
	}
	if n < 0 {
		panic(vm.errf("field: negative index %d", n))
	}
	if n >= int64(len(t)) {
		panic(vm.errf("field: index %d >= length %d", n, len(t)))
	}
	vm.push(datastack, t[n])
}
