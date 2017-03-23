package vm

var InitialRunLimit = initialRunLimit

type VirtualMachine struct {
	Program           []byte
	RunLimit          int64
	DataStack         [][]byte
	DeferredCost      int64
	Context           *Context
	PC                uint32
	NextPC            uint32
	Data              []byte
	ExpansionReserved bool
}

func VMtovm(in *VirtualMachine) *virtualMachine {
	dataStack := make([][]byte, len(in.DataStack))
	copy(dataStack, in.DataStack)
	return &virtualMachine{
		program:           in.Program,
		runLimit:          in.RunLimit,
		dataStack:         dataStack,
		deferredCost:      in.DeferredCost,
		context:           in.Context,
		pc:                in.PC,
		nextPC:            in.NextPC,
		data:              in.Data,
		expansionReserved: in.ExpansionReserved,
	}
}

func VMfromvm(in *virtualMachine) *VirtualMachine {
	return &VirtualMachine{
		Program:           in.program,
		RunLimit:          in.runLimit,
		DataStack:         in.dataStack,
		DeferredCost:      in.deferredCost,
		Context:           in.context,
		PC:                in.pc,
		NextPC:            in.nextPC,
		Data:              in.data,
		ExpansionReserved: in.expansionReserved,
	}
}

func (vm *VirtualMachine) Run() (*VirtualMachine, error) {
	realVM := VMtovm(vm)
	err := realVM.run()
	return VMfromvm(realVM), err
}

func (vm *VirtualMachine) Step() (*VirtualMachine, error) {
	realVM := VMtovm(vm)
	err := realVM.step()
	return VMfromvm(realVM), err
}

func (vm *VirtualMachine) FalseResult() bool {
	realVM := VMtovm(vm)
	return realVM.falseResult()
}

func OpName(op Op) string {
	return ops[op].name
}

func CallOp(op Op, vm *VirtualMachine) (*VirtualMachine, error) {
	realVM := VMtovm(vm)
	err := ops[op].fn(realVM)
	return VMfromvm(realVM), err
}
