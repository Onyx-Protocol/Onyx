package vm

import "chain-stealth/math/checked"

func opCat(vm *virtualMachine) error {
	err := vm.applyCost(4)
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
	lens := int64(len(a) + len(b))
	err = vm.applyCost(lens)
	if err != nil {
		return err
	}
	vm.deferCost(-lens)
	err = vm.push(append(a, b...), true)
	if err != nil {
		return err
	}
	return nil
}

func opSubstr(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	size, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if size < 0 {
		return ErrBadValue
	}
	err = vm.applyCost(size)
	if err != nil {
		return err
	}
	vm.deferCost(-size)
	offset, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if offset < 0 {
		return ErrBadValue
	}
	str, err := vm.pop(true)
	if err != nil {
		return err
	}
	end, ok := checked.AddInt64(offset, size)
	if !ok || end > int64(len(str)) {
		return ErrBadValue
	}
	err = vm.push(str[offset:end], true)
	if err != nil {
		return err
	}
	return nil
}

func opLeft(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	size, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if size < 0 {
		return ErrBadValue
	}
	err = vm.applyCost(size)
	if err != nil {
		return err
	}
	vm.deferCost(-size)
	str, err := vm.pop(true)
	if err != nil {
		return err
	}
	if size > int64(len(str)) {
		return ErrBadValue
	}
	err = vm.push(str[:size], true)
	if err != nil {
		return err
	}
	return nil
}

func opRight(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	size, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if size < 0 {
		return ErrBadValue
	}
	err = vm.applyCost(size)
	if err != nil {
		return err
	}
	vm.deferCost(-size)
	str, err := vm.pop(true)
	if err != nil {
		return err
	}
	lstr := int64(len(str))
	if size > lstr {
		return ErrBadValue
	}
	err = vm.push(str[lstr-size:], true)
	if err != nil {
		return err
	}
	return nil
}

func opSize(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	str, err := vm.top()
	if err != nil {
		return err
	}
	err = vm.pushInt64(int64(len(str)), true)
	if err != nil {
		return err
	}
	return nil
}

func opCatpushdata(vm *virtualMachine) error {
	err := vm.applyCost(4)
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
	lb := len(b)
	lens := int64(len(a) + lb)
	err = vm.applyCost(lens)
	if err != nil {
		return err
	}
	vm.deferCost(-lens)
	return vm.push(append(a, PushdataBytes(b)...), true)
}
