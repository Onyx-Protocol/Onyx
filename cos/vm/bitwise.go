package vm

import "bytes"

func opInvert(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	top, err := vm.top()
	if err != nil {
		return err
	}
	err = vm.applyCost(int64(len(top)))
	if err != nil {
		return err
	}
	// Could rewrite top in place but maybe it's a shared data
	// structure?
	newTop := make([]byte, 0, len(top))
	for _, b := range top {
		newTop = append(newTop, ^b)
	}
	vm.dataStack[len(vm.dataStack)-1] = newTop
	return nil
}

func opAnd(vm *virtualMachine) error {
	err := vm.applyCost(1)
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
	min, max := len(a), len(b)
	if min > max {
		min, max = max, min
	}
	err = vm.applyCost(int64(min))
	if err != nil {
		return err
	}
	res := make([]byte, 0, min)
	for i := 0; i < min; i++ {
		res = append(res, a[i]&b[i])
	}
	return vm.push(res, true)
}

func opOr(vm *virtualMachine) error {
	return doOr(vm, false)
}

func opXor(vm *virtualMachine) error {
	return doOr(vm, true)
}

func doOr(vm *virtualMachine, xor bool) error {
	err := vm.applyCost(1)
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
	min, max := len(a), len(b)
	if min > max {
		min, max = max, min
	}
	err = vm.applyCost(int64(max))
	if err != nil {
		return err
	}
	res := make([]byte, 0, max)
	for i := 0; i < max; i++ {
		var aByte, bByte, resByte byte
		if i >= len(a) {
			aByte = 0
		} else {
			aByte = a[i]
		}
		if i >= len(b) {
			bByte = 0
		} else {
			bByte = b[i]
		}
		if xor {
			resByte = aByte ^ bByte
		} else {
			resByte = aByte | bByte
		}

		res = append(res, resByte)
	}
	return vm.push(res, true)
}

func opEqual(vm *virtualMachine) error {
	res, err := doEqual(vm)
	if err != nil {
		return err
	}
	return vm.pushBool(res, true)
}

func opEqualVerify(vm *virtualMachine) error {
	res, err := doEqual(vm)
	if err != nil {
		return err
	}
	if res {
		return nil
	}
	return ErrVerifyFailed
}

func doEqual(vm *virtualMachine) (bool, error) {
	err := vm.applyCost(1)
	if err != nil {
		return false, err
	}
	b, err := vm.pop(true)
	if err != nil {
		return false, err
	}
	a, err := vm.pop(true)
	if err != nil {
		return false, err
	}
	min, max := len(a), len(b)
	if min > max {
		min, max = max, min
	}
	err = vm.applyCost(int64(min))
	if err != nil {
		return false, err
	}
	return bytes.Equal(a, b), nil
}
