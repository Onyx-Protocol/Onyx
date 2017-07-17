package txvm2

func opTuple(vm *vm) {
	n := vm.popInt64(datastack)
	var vals []value
	for n > 0 {
		v := vm.pop(datastack)
		vals = append(vals, v)
		n--
	}
	vm.push(tuple(vals))
}

func opUntuple(vm *vm) {
	v := vm.pop(datastack)
	t, ok := v.(tuple)
	if !ok {
		panic(xxx)
	}
	for i := len(t) - 1; i >= 0; i-- {
		vm.push(t[i])
	}
	vm.push(vint64(len(t)))
}

func opField(vm *vm) {
	n := vm.popInt64()
	v := vm.pop()
	t, ok := v.(tuple)
	if !ok {
		panic(xxx)
	}
	if n < 0 {
		panic(xxx)
	}
	if n >= len(t) {
		panic(xxx)
	}
	vm.push(t[n])
}
