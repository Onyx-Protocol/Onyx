package txvm2

import "bytes"

func opRoll(vm *vm) {
	stackID := vm.popInt64()
	switch stackID {
	case commandstack, effectstack:
		// xxx error
	}
	s := vm.getStack(stackID)
	n := vm.popInt64()
	s.roll(n)
	// xxx runlimit
}

func opBury(vm *vm) {
	stackID := vm.popInt64()
	switch stackID {
	case commandstack, effectstack:
		// xxx error
	}
	s := vm.getStack(stackID)
	n := vm.popInt64()
	s.bury(n)
	// xxx runlimit
}

func opReverse(vm *vm) {
	stackID := vm.popInt64()
	switch stackID {
	case commandstack, effectstack:
		// xxx error
	}
	s := vm.getStack(stackID)
	n := vm.popInt64()
	vals := s.popN(n)
	s.pushN(vals)
	// xxx runlimit
}

func opDepth(vm *vm) {
	stackID := vm.popInt64()
	s := vm.getStack(stackID)
	vm.pushInt64(s.depth())
	// xxx runlimit
}

func opPeek(vm *vm) {
	stackID := vm.popInt64()
	s := vm.getStack(stackID)
	n := vm.popInt64()
	vm.push(s.peek(n))
}

func opEqual(vm *vm) {
	v1 := vm.pop()
	v2 := vm.pop()
	t1 := v1.typ()
	t2 := v2.typ()
	res := false
	if t1 == t2 && t1 != tupleType {
		switch t1 {
		case int64type:
			res = v1.(vint64) == v2.(vint64)
		case stringtype:
			res = bytes.Equal(v1.(vstring), v2.(vstring))
		}
	}
	vm.pushBool(res)
}

func opType(vm *vm) {
	v := vm.pop()
	vm.pushInt64(v.typ())
}

func opLen(vm *vm) {
	v := vm.pop()
	switch v := v.(type) {
	case vstring:
		vm.pushInt64(len(v))
	case tuple:
		vm.pushInt64(len(v))
	default:
		panic(xxx)
	}
}

func opDrop(vm *vm) {
	vm.pop()
}

func opToAlt(vm *vm) {
	v := vm.pop()
	vm.stacks[altstack].push(v)
}

func opFromAlt(vm *vm) {
	v, ok := vm.stacks[altstack].pop()
	if !ok {
		panic(xxx)
	}
	vm.push(v)
}
