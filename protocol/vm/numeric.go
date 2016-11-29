package vm

import (
	"math"

	"chain-stealth/math/checked"
)

func op1Add(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.AddInt64(n, 1)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func op1Sub(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.SubInt64(n, 1)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func op2Mul(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.MulInt64(n, 2)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func op2Div(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	return vm.pushInt64(n>>1, true)
}

func opNegate(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.NegateInt64(n)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func opAbs(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if n == math.MinInt64 {
		return ErrRange
	}
	if n < 0 {
		n = -n
	}
	return vm.pushInt64(n, true)
}

func opNot(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	return vm.pushBool(n == 0, true)
}

func op0NotEqual(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	return vm.pushBool(n != 0, true)
}

func opAdd(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.AddInt64(x, y)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func opSub(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.SubInt64(x, y)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func opMul(vm *virtualMachine) error {
	err := vm.applyCost(8)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	res, ok := checked.MulInt64(x, y)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func opDiv(vm *virtualMachine) error {
	err := vm.applyCost(8)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if y == 0 {
		return ErrDivZero
	}
	res, ok := checked.DivInt64(x, y)
	if !ok {
		return ErrRange
	}
	return vm.pushInt64(res, true)
}

func opMod(vm *virtualMachine) error {
	err := vm.applyCost(8)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if y == 0 {
		return ErrDivZero
	}

	res, ok := checked.ModInt64(x, y)
	if !ok {
		return ErrRange
	}

	// Go's modulus operator produces the wrong result for mixed-sign
	// operands
	if res != 0 && (x >= 0) != (y >= 0) {
		res += y
	}

	return vm.pushInt64(res, true)
}

func opLshift(vm *virtualMachine) error {
	err := vm.applyCost(8)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if y < 0 {
		return ErrBadValue
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if x == 0 || y == 0 {
		return vm.pushInt64(x, true)
	}

	res, ok := checked.LshiftInt64(x, y)
	if !ok {
		return ErrRange
	}

	return vm.pushInt64(res, true)
}

func opRshift(vm *virtualMachine) error {
	err := vm.applyCost(8)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if y < 0 {
		return ErrBadValue
	}
	return vm.pushInt64(x>>uint64(y), true)
}

func opBoolAnd(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	b, err := vm.pop(true)
	if err != nil {
		return err
	}
	a, err := vm.pop(true)
	if err != nil {
		return err
	}
	return vm.pushBool(AsBool(a) && AsBool(b), true)
}

func opBoolOr(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	b, err := vm.pop(true)
	if err != nil {
		return err
	}
	a, err := vm.pop(true)
	if err != nil {
		return err
	}
	return vm.pushBool(AsBool(a) || AsBool(b), true)
}

const (
	cmpLess = iota
	cmpLessEqual
	cmpGreater
	cmpGreaterEqual
	cmpEqual
	cmpNotEqual
)

func opNumEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpEqual)
}

func opNumEqualVerify(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if x == y {
		return nil
	}
	return ErrVerifyFailed
}

func opNumNotEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpNotEqual)
}

func opLessThan(vm *virtualMachine) error {
	return doNumCompare(vm, cmpLess)
}

func opGreaterThan(vm *virtualMachine) error {
	return doNumCompare(vm, cmpGreater)
}

func opLessThanOrEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpLessEqual)
}

func opGreaterThanOrEqual(vm *virtualMachine) error {
	return doNumCompare(vm, cmpGreaterEqual)
}

func doNumCompare(vm *virtualMachine, op int) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	var res bool
	switch op {
	case cmpLess:
		res = x < y
	case cmpLessEqual:
		res = x <= y
	case cmpGreater:
		res = x > y
	case cmpGreaterEqual:
		res = x >= y
	case cmpEqual:
		res = x == y
	case cmpNotEqual:
		res = x != y
	}
	return vm.pushBool(res, true)
}

func opMin(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if x > y {
		x = y
	}
	return vm.pushInt64(x, true)
}

func opMax(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	y, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if x < y {
		x = y
	}
	return vm.pushInt64(x, true)
}

func opWithin(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	max, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	min, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	x, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	return vm.pushBool(x >= min && x < max, true)
}
