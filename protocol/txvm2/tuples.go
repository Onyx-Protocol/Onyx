package txvm2

import "fmt"

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
		panic(fmt.Errorf("untuple: %T is not a tuple", v))
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
		panic(fmt.Errorf("field: %T is not a tuple", v))
	}
	if n < 0 {
		panic(fmt.Errorf("field: negative index %d", n))
	}
	if n >= len(t) {
		panic(fmt.Errorf("field: index %d >= length %d", n, len(t)))
	}
	vm.push(t[n])
}
