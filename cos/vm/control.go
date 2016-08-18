package vm

type cfType uint8

const (
	cfIf cfType = iota
	cfElse
	cfWhile
)

type controlTuple struct {
	optype cfType
	flag   bool
	pc     uint32
}

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
	if len(vm.controlStack) > 0 && !vm.controlStack[len(vm.controlStack)-1].flag {
		// skip
		vm.controlStack = append(vm.controlStack, controlTuple{optype: cfIf, flag: false})
	} else {
		// execute
		p, err := vm.pop(true)
		if err != nil {
			return err
		}
		vm.controlStack = append(vm.controlStack, controlTuple{optype: cfIf, flag: AsBool(p) != negate})
	}
	return nil
}

func opElse(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	if len(vm.controlStack) == 0 {
		return ErrControlStackUnderflow
	}
	v := popControl(vm)
	if v.optype != cfIf {
		return ErrBadControlSyntax
	}
	if len(vm.controlStack) > 0 && !vm.controlStack[len(vm.controlStack)-1].flag {
		// skip
		vm.controlStack = append(vm.controlStack, controlTuple{optype: cfElse, flag: false})
		return nil
	}
	vm.controlStack = append(vm.controlStack, controlTuple{optype: cfElse, flag: !v.flag})
	return nil
}

func opEndif(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if len(vm.controlStack) == 0 {
		return ErrControlStackUnderflow
	}
	v := popControl(vm)
	if v.optype != cfIf && v.optype != cfElse {
		return ErrBadControlSyntax
	}
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

func opReturn(vm *virtualMachine) error {
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
	vm.deferCost(-stackCost(childVM.dataStack) + preStackCost)
	vm.deferCost(-stackCost(childVM.altStack))

	err = vm.pushBool(childErr == nil && ok, true)
	if err != nil {
		return err
	}
	return nil
}

func opWhile(vm *virtualMachine) error {
	err := vm.applyCost(4)
	if err != nil {
		return err
	}
	if len(vm.controlStack) > 0 && !vm.controlStack[len(vm.controlStack)-1].flag {
		// skip
		vm.controlStack = append(vm.controlStack, controlTuple{optype: cfWhile, flag: false})
		return nil
	}
	val, err := vm.top()
	if err != nil {
		return err
	}
	vm.controlStack = append(vm.controlStack, controlTuple{optype: cfWhile, flag: AsBool(val), pc: vm.pc})
	if !AsBool(val) {
		vm.pop(true)
	}
	return nil
}

func opEndwhile(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if len(vm.controlStack) == 0 {
		return ErrControlStackUnderflow
	}
	v := popControl(vm)
	if v.optype != cfWhile {
		return ErrBadControlSyntax
	}
	if v.flag {
		vm.nextPC = v.pc
	}
	return nil
}

func popControl(vm *virtualMachine) (cf controlTuple) {
	cf, vm.controlStack = vm.controlStack[len(vm.controlStack)-1], vm.controlStack[:len(vm.controlStack)-1]
	return
}
