package vm

func opIf(vm *virtualMachine) error {
	return doIf(vm, false)
}

func opNotIf(vm *virtualMachine) error {
	return doIf(vm, true)
}

func doIf(vm *virtualMachine, negate bool) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	if len(vm.condStack) > 0 && !vm.condStack[len(vm.condStack)-1] {
		// skip
		vm.condStack = append(vm.condStack, false)
	} else {
		// execute
		p, err := vm.pop(true)
		if err != nil {
			return err
		}
		vm.condStack = append(vm.condStack, AsBool(p) != negate)
	}
	return nil
}

func opElse(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	if len(vm.condStack) == 0 {
		return ErrCondStackUnderflow
	}
	v := vm.condStack[len(vm.condStack)-1]
	vm.condStack = append(vm.condStack[:len(vm.condStack)-1], !v)
	return nil
}

func opEndif(vm *virtualMachine) error {
	if len(vm.condStack) == 0 {
		return ErrCondStackUnderflow
	}
	vm.condStack = vm.condStack[:len(vm.condStack)-1]
	return nil
}

func opVerify(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	p, err := vm.pop(true)
	if err != nil {
		return err
	}
	if AsBool(p) {
		return nil
	}
	return ErrVerifyFailed
}

func opReturn(_ *virtualMachine) error {
	return ErrReturn
}

func opCheckPredicate(vm *virtualMachine) error {
	err := vm.applyCost(256)
	if err != nil {
		return err
	}
	vm.deferCost(-256 + 64) // get most of that cost back at the end
	limit, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	predicate, err := vm.pop(true)
	if err != nil {
		return err
	}
	if limit < 0 {
		return ErrBadValue
	}
	if limit == 0 {
		limit = vm.runLimit
	}
	err = vm.applyCost(limit)
	if err != nil {
		return err
	}
	childVM := virtualMachine{
		program:    predicate,
		runLimit:   limit,
		depth:      vm.depth + 1,
		dataStack:  append([][]byte{}, vm.dataStack...),
		tx:         vm.tx,
		inputIndex: vm.inputIndex,
		traceOut:   vm.traceOut,
	}
	preStackCost := stackCost(childVM.dataStack)
	ok, childErr := childVM.run()

	vm.deferCost(-childVM.runLimit)
	vm.deferCost(stackCost(childVM.dataStack) - preStackCost)
	vm.deferCost(-stackCost(childVM.altStack))

	err = vm.pushBool(childErr == nil && ok, true)
	if err != nil {
		return err
	}
	return nil
}

func opWhile(vm *virtualMachine) error {
	if len(vm.condStack) > 0 && !vm.condStack[len(vm.condStack)-1] {
		// skip
		vm.condStack = append(vm.condStack, false)
		return nil
	}
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	val, err := vm.top()
	if err != nil {
		return err
	}
	vm.condStack = append(vm.condStack, AsBool(val))
	if AsBool(val) {
		vm.loopStack = append(vm.loopStack, vm.pc)
		return nil
	}
	vm.pop(true)
	return nil
}

func opEndwhile(vm *virtualMachine) error {
	if len(vm.condStack) == 0 {
		return ErrCondStackUnderflow
	}
	if len(vm.loopStack) == 0 {
		return ErrLoopStackUnderflow
	}
	c := vm.condStack[len(vm.condStack)-1]
	vm.condStack = vm.condStack[:len(vm.condStack)-1]
	if c {
		vm.loopStack = vm.loopStack[:len(vm.loopStack)-1]
		return nil
	}
	return nil
}
