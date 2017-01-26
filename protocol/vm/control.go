package vm

import (
	"encoding/binary"
)

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

func opFail(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
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
	n, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if limit < 0 {
		return ErrBadValue
	}
	l := int64(len(vm.dataStack))
	if n > l {
		return ErrDataStackUnderflow
	}
	if limit == 0 {
		limit = vm.runLimit
	}
	err = vm.applyCost(limit)
	if err != nil {
		return err
	}

	childVM := virtualMachine{
		mainprog:   vm.mainprog,
		program:    predicate,
		runLimit:   limit,
		depth:      vm.depth + 1,
		dataStack:  append([][]byte{}, vm.dataStack[l-n:]...),
		tx:         vm.tx,
		inputIndex: vm.inputIndex,
		sigHasher:  vm.sigHasher,
	}
	vm.dataStack = vm.dataStack[:l-n]

	ok, childErr := childVM.run()

	vm.deferCost(-childVM.runLimit)
	vm.deferCost(-stackCost(childVM.dataStack))
	vm.deferCost(-stackCost(childVM.altStack))

	err = vm.pushBool(childErr == nil && ok, true)
	if err != nil {
		return err
	}
	return nil
}

func opJump(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	address := binary.LittleEndian.Uint32(vm.data)
	vm.nextPC = address
	return nil
}

func opJumpIf(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	p, err := vm.pop(true)
	if err != nil {
		return err
	}
	if AsBool(p) {
		address := binary.LittleEndian.Uint32(vm.data)
		vm.nextPC = address
	}
	return nil
}
