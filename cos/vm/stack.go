package vm

func opToAltStack(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	if len(vm.dataStack) == 0 {
		return ErrDataStackUnderflow
	}
	// no standard memory cost accounting here
	vm.altStack = append(vm.altStack, vm.dataStack[len(vm.dataStack)-1])
	vm.dataStack = vm.dataStack[:len(vm.dataStack)-1]
	return nil
}

func opFromAltStack(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	if len(vm.altStack) == 0 {
		return ErrAltStackUnderflow
	}
	// no standard memory cost accounting here
	vm.dataStack = append(vm.dataStack, vm.altStack[len(vm.altStack)-1])
	vm.altStack = vm.altStack[:len(vm.altStack)-1]
	return nil
}

func op2Drop(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		_, err = vm.pop(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func op2Dup(vm *virtualMachine) error {
	return nDup(vm, 2)
}

func op3Dup(vm *virtualMachine) error {
	return nDup(vm, 3)
}

func nDup(vm *virtualMachine, n int) error {
	err := vm.applyCost(int64(n))
	if err != nil {
		return err
	}
	if len(vm.dataStack) < n {
		return ErrDataStackUnderflow
	}
	for i := 0; i < n; i++ {
		err = vm.push(vm.dataStack[len(vm.dataStack)-n], false)
		if err != nil {
			return err
		}
	}
	return nil
}

func op2Over(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	if len(vm.dataStack) < 4 {
		return ErrDataStackUnderflow
	}
	for i := 0; i < 2; i++ {
		err = vm.push(vm.dataStack[len(vm.dataStack)-4], false)
		if err != nil {
			return err
		}
	}
	return nil
}

func op2Rot(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	if len(vm.dataStack) < 6 {
		return ErrDataStackUnderflow
	}
	newStack := make([][]byte, 0, len(vm.dataStack))
	newStack = append(newStack, vm.dataStack[:len(vm.dataStack)-6]...)
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-4:]...)
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-6])
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-5])
	vm.dataStack = newStack
	return nil
}

func op2Swap(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	if len(vm.dataStack) < 4 {
		return ErrDataStackUnderflow
	}
	newStack := make([][]byte, 0, len(vm.dataStack))
	newStack = append(newStack, vm.dataStack[:len(vm.dataStack)-4]...)
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-2:]...)
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-4])
	newStack = append(newStack, vm.dataStack[len(vm.dataStack)-3])
	vm.dataStack = newStack
	return nil
}

func opIfDup(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	item, err := vm.top()
	if err != nil {
		return err
	}
	if AsBool(item) {
		err = vm.push(item, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func opDepth(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	err = vm.pushInt64(int64(len(vm.dataStack)), false)
	if err != nil {
		return err
	}
	return nil
}

func opDrop(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	_, err = vm.pop(false)
	if err != nil {
		return err
	}
	return nil
}

func opDup(vm *virtualMachine) error {
	return nDup(vm, 1)
}

func opNip(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	top, err := vm.top()
	if err != nil {
		return err
	}
	// temporarily pop off the top value with no standard memory accounting
	vm.dataStack = vm.dataStack[:len(vm.dataStack)-1]
	_, err = vm.pop(false)
	if err != nil {
		return err
	}
	// now put the top item back
	vm.dataStack = append(vm.dataStack, top)
	return nil
}

func opOver(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if len(vm.dataStack) < 2 {
		return ErrDataStackUnderflow
	}
	err = vm.push(vm.dataStack[len(vm.dataStack)-2], false)
	if err != nil {
		return err
	}
	return nil
}

func opPick(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(false)
	if err != nil {
		return err
	}
	if int64(len(vm.dataStack)) < n+1 {
		return ErrDataStackUnderflow
	}
	err = vm.push(vm.dataStack[int64(len(vm.dataStack))-(n+1)], false)
	if err != nil {
		return err
	}
	return nil
}

func opRoll(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	n, err := vm.popInt64(false)
	if err != nil {
		return err
	}
	// TODO(bobg): range-check n
	err = rot(vm, n+1)
	if err != nil {
		return err
	}
	return nil
}

func opRot(vm *virtualMachine) error {
	err := vm.applyCost(2)
	if err != nil {
		return err
	}
	err = rot(vm, 3)
	if err != nil {
		return err
	}
	return nil
}

func rot(vm *virtualMachine, n int64) error {
	if int64(len(vm.dataStack)) < n {
		return ErrDataStackUnderflow
	}
	index := int64(len(vm.dataStack)) - n
	newStack := make([][]byte, 0, len(vm.dataStack))
	newStack = append(newStack, vm.dataStack[:index]...)
	newStack = append(newStack, vm.dataStack[index+1:]...)
	newStack = append(newStack, vm.dataStack[index])
	vm.dataStack = newStack
	return nil
}

func opSwap(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	l := len(vm.dataStack)
	if l < 2 {
		return ErrDataStackUnderflow
	}
	vm.dataStack[l-1], vm.dataStack[l-2] = vm.dataStack[l-2], vm.dataStack[l-1]
	return nil
}

func opTuck(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if len(vm.dataStack) < 2 {
		return ErrDataStackUnderflow
	}
	top2 := vm.dataStack[len(vm.dataStack)-2:]
	// temporarily remove the top two items without standard memory accounting
	vm.dataStack = vm.dataStack[:len(vm.dataStack)-2]
	err = vm.push(top2[1], false)
	if err != nil {
		return err
	}
	vm.dataStack = append(vm.dataStack, top2...)
	return nil
}
