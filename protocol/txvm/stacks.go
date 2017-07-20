package txvm

import "bytes"

func opRoll(vm *vm) {
	stackID := vm.popInt64(datastack)
	switch stackID {
	case commandstack, effectstack:
		panic(vm.errf("cannot roll on stack %d", stackID))
	}
	s := vm.stacks[stackID] // xxx range check
	n := vm.popInt64(datastack)
	s.roll(int64(n))
	// xxx runlimit
}

func opBury(vm *vm) {
	stackID := vm.popInt64(datastack)
	switch stackID {
	case commandstack, effectstack:
		panic(vm.errf("cannot bury on stack %d", stackID))
	}
	s := vm.stacks[stackID] // xxx range check
	n := vm.popInt64(datastack)
	s.bury(int64(n))
	// xxx runlimit
}

func opReverse(vm *vm) {
	stackID := vm.popInt64(datastack)
	switch stackID {
	case commandstack, effectstack:
		panic(vm.errf("cannot reverse on stack %d", stackID))
	}
	s := vm.stacks[stackID] // xxx range check
	n := vm.popInt64(datastack)
	vals := s.popN(int64(n))
	if int64(len(vals)) != int64(n) {
		panic(vm.errf("too few items on stack (%d vs. %d)", len(vals), n))
	}
	s.pushN(vals)
	// xxx runlimit
}

func opDepth(vm *vm) {
	stackID := vm.popInt64(datastack)
	s := vm.getStack(int64(stackID))
	vm.push(datastack, vint64(len(*s)))
	// xxx runlimit
}

func opPeek(vm *vm) {
	stackID := vm.popInt64(datastack)
	s := vm.getStack(int64(stackID))
	n := vm.popInt64(datastack)
	item, ok := s.peek(int64(n))
	if !ok {
		panic(vm.errf("too few items on stack (%d vs.  %d)", len(*s), n))
	}
	vm.push(datastack, item)
}

func opEqual(vm *vm) {
	v1 := vm.pop(datastack)
	v2 := vm.pop(datastack)
	t1 := v1.typ()
	t2 := v2.typ()
	res := false
	if t1 == t2 && t1 != tupletype {
		switch t1 {
		case int64type:
			res = v1.(vint64) == v2.(vint64)
		case bytestype:
			res = bytes.Equal(v1.(vbytes), v2.(vbytes))
		}
	}
	vm.pushBool(datastack, res)
}

func opType(vm *vm) {
	v := vm.pop(datastack)
	vm.push(datastack, vint64(v.typ()))
}

func opLen(vm *vm) {
	v := vm.pop(datastack)
	switch v := v.(type) {
	case vbytes:
		vm.push(datastack, vint64(len(v)))
	case tuple:
		vm.push(datastack, vint64(len(v)))
	default:
		panic(vm.errf("len: cannot take the length of %T", v))
	}
}

func opDrop(vm *vm) {
	vm.pop(datastack)
}

func opToAlt(vm *vm) {
	v := vm.pop(datastack)
	vm.push(altstack, v)
}

func opFromAlt(vm *vm) {
	v := vm.pop(altstack)
	vm.push(datastack, v)
}
