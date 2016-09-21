package vm

import "math"

func op1Add(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if n == math.MaxInt64 {
		return ErrRange
	}
	return vm.pushInt64(n+1, true)
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
	if n == math.MinInt64 {
		return ErrRange
	}
	return vm.pushInt64(n-1, true)
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
	res, ok := multiplyCheckOverflow(n, 2)
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
	if n == math.MinInt64 {
		return ErrRange
	}
	return vm.pushInt64(-n, true)
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
	res, ok := addCheckOverflow(x, y)
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
	res, ok := subCheckOverflow(x, y)
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
	res, ok := multiplyCheckOverflow(x, y)
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
	res, ok := divideCheckOverflow(x, y)
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

	res, ok := modCheckOverflow(x, y)
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

	var maxShift int64
	if x < 0 {
		maxShift = 64
	} else {
		maxShift = 63
	}
	if y >= maxShift {
		return ErrRange
	}

	// Check for this separately since we can't take the abs of MinInt64.
	if x == int64(math.MinInt64) {
		return ErrRange
	}

	// How far can we left shift? Look for the most significant bit.
	var absX, msb int64
	if x < 0 {
		absX = -x
	} else {
		absX = x
	}
	for absX > 0 {
		msb++
		absX >>= 1
	}
	if y > maxShift-msb {
		return ErrRange
	}

	return vm.pushInt64(x<<uint64(y), true)
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

func addCheckOverflow(a, b int64) (sum int64, ok bool) {
	if (b > 0 && a > math.MaxInt64-b) ||
		(b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}

func subCheckOverflow(a, b int64) (diff int64, ok bool) {
	if (b > 0 && a < math.MinInt64+b) ||
		(b < 0 && a > math.MaxInt64+b) {
		return 0, false
	}
	return a - b, true
}

func multiplyCheckOverflow(a, b int64) (product int64, ok bool) {
	if (a > 0 && b > 0 && a > math.MaxInt64/b) ||
		(a > 0 && b <= 0 && b < math.MinInt64/a) ||
		(a <= 0 && b > 0 && a < math.MinInt64/b) ||
		(a < 0 && b <= 0 && b < math.MaxInt64/a) {
		return 0, false
	}
	return a * b, true
}

func divideCheckOverflow(a, b int64) (quotient int64, ok bool) {
	if b == 0 || (a == math.MinInt64 && b == -1) {
		return 0, false
	}
	return a / b, true
}

func modCheckOverflow(a, b int64) (remainder int64, ok bool) {
	if b == 0 || (a == math.MinInt64 && b == -1) {
		return 0, false
	}
	return a % b, true
}
