package vm

import "chain/protocol/bc"

type VM struct {
	Program           []byte
	RunLimit          int
	DataStack         [][]byte
	PC                uint32
	NextPC            uint32
	Data              []byte
	DeferredCost      int64
	TX                *bc.Tx
	ExpansionReserved bool
}

func NewVM(vm *VM) *virtualMachine {
	return &virtualMachine{
		program:           vm.Program,
		runLimit:          int64(vm.RunLimit),
		dataStack:         vm.DataStack,
		pc:                vm.PC,
		nextPC:            vm.NextPC,
		data:              vm.Data,
		deferredCost:      vm.DeferredCost,
		tx:                vm.TX,
		expansionReserved: vm.ExpansionReserved,
	}
}

func RunVM(vm *virtualMachine) error {
	return vm.run()
}

func StepVM(vm *virtualMachine) error {
	return vm.step()
}

func IsFalseResult(vm *virtualMachine) bool {
	return vm.falseResult()
}

var InitialRunLimit = initialRunLimit
