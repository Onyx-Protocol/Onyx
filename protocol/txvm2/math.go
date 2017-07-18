package txvm2

import "chain/math/checked"

func opAdd(vm *vm) {
	binOp(vm, checked.AddInt64)
}

func opSub(vm *vm) {
	binOp(vm, checked.SubInt64)
}

func opMul(vm *vm) {
	binOp(vm, checked.MulInt64)
}

func opDiv(vm *vm) {
	binOp(vm, checked.DivInt64)
}

func opMod(vm *vm) {
	binOp(vm, checked.ModInt64)
}

func opLeftShift(vm *vm) {
	binOp(vm, checked.LshiftInt64)
}

func opRightShift(vm *vm) {
	binOp(vm, checked.RshiftInt64)
}

func binOp(vm *vm, f func(a, b int64) (int64, bool)) {
	a := vm.popInt64(datastack)
	b := vm.popInt64(datastack)
	res, ok := f(int64(a), int64(b))
	if !ok {
		panic("arithmetic overflow")
	}
	vm.push(datastack, vint64(res))
}

func opGreaterThan(vm *vm) {
	a := vm.popInt64(datastack)
	b := vm.popInt64(datastack)
	vm.pushBool(datastack, a > b)
}
