package txvm2

import (
	"bytes"
	"fmt"
)

func opRoll(vm *vm) {
	stackID := vm.popInt64(datastack)
	switch stackID {
	case commandstack, effectstack:
		panic(fmt.Errorf("cannot roll on stack %d", stackID))
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
		panic(fmt.Errorf("cannot bury on stack %d", stackID))
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
		panic(fmt.Errorf("cannot reverse on stack %d", stackID))
	}
	s := vm.stacks[stackID] // xxx range check
	n := vm.popInt64(datastack)
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
		case bytestype:
			res = bytes.Equal(v1.(vbytes), v2.(vbytes))
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
	case vbytes:
		vm.pushInt64(len(v))
	case tuple:
		vm.pushInt64(len(v))
	default:
		panic(fmt.Errorf("len: cannot take the length of %T", v))
	}
}

func opDrop(vm *vm) {
	vm.pop()
}

func opToAlt(vm *vm) {
	v := vm.pop(datastack)
	vm.push(altstack, v)
}

func opFromAlt(vm *vm) {
	v := vm.pop(altstack)
	vm.push(datastack, v)
}
