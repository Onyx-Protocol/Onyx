package txvm2

func opTuple(vm *vm) {
	n := vm.popInt64()
	var vals []value
	for n > 0 {
		v := vm.pop()
		vals = append(vals, v)
		n--
	}
	vm.push(tuple(vals))
}

func opUntuple(vm *vm) {
	t := vm.popTuple()
	for i := len(t) - 1; i >= 0; i-- {
		vm.push(t[i])
	}
	vm.pushInt64(len(t))
}

func opField(vm *vm) {
	n := vm.popInt64()
	t := vm.popTuple()
	if n < 0 {
		panic(xxx)
	}
	if n >= len(t) {
		panic(xxx)
	}
	vm.push(t[n])
}
