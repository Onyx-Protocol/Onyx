package txvm2

import "chain/math/checked"

func opAdd(vm *vm) {
	binary(vm, checked.AddInt64)
}

func opSub(vm *vm) {
	binary(vm, checked.SubInt64)
}

func opMul(vm *vm) {
	binary(vm, checked.MulInt64)
}

func opDiv(vm *vm) {
	binary(vm, checked.DivInt64)
}

func opMod(vm *vm) {
	binary(vm, checked.ModInt64)
}

func opLeftShift(vm *vm) {
	binary(vm, checked.LshiftInt64)
}

func opRightShift(vm *vm) {
	binary(vm, checked.RshiftInt64)
}

func binary(vm *vm, f func(a, b int64) (int64, bool)) {
	a := vm.popInt64()
	b := vm.popInt64()
	res, ok := f(a, b)
	if !ok {
		panic(xxx)
	}
	vm.pushInt64(res)
}

func opGreaterThan(vm *vm) {
	a := vm.popInt64()
	b := vm.popInt64()
	vm.pushBool(a > b)
}
