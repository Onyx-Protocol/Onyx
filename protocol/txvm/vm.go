package txvm

import (
	"encoding/binary"
	"errors"
)

type vm struct {
	// config, doesn't change after init
	traceUnlock func(Contract)
	traceLock   func(Contract)
	traceOp     func(stack, byte, []byte, []byte)
	traceError  func(error)
	tmin        int64
	tmax        int64

	pc   int    // program counter
	prog []byte // current program

	data stack
	alt  stack

	inputs          tupleStack
	values          tupleStack
	outputs         tupleStack
	conditions      tupleStack
	nonces          tupleStack
	anchors         tupleStack
	retirements     tupleStack
	timeconstraints tupleStack
	annotations     tupleStack
	summary         tupleStack
}

// Validate returns whether x is valid.
//
// To get detailed information about a Tx,
// such as determining why an invalid Tx is invalid,
// use Option funcs to trace execution.
func Validate(tx []byte, o ...Option) bool {
	vm := &vm{
		traceUnlock: func(Contract) {},
		traceLock:   func(Contract) {},
	}
	for _, o := range o {
		o(vm)
	}

	defer func() {
		err := recover()
		if err, ok := err.(error); ok {
			vm.traceError(err)
		}
	}()

	exec(vm, tx)

	// TODO(kr): call some tracing hook here
	// to signal end of execution.

	return vm.summary.Len() == 1 &&
		vm.inputs.Len() == 0 &&
		vm.values.Len() == 0 &&
		vm.outputs.Len() == 0 &&
		vm.conditions.Len() == 0 &&
		vm.nonces.Len() == 0 &&
		vm.retirements.Len() == 0 &&
		vm.timeconstraints.Len() == 0 &&
		vm.annotations.Len() == 0
}

func exec(vm *vm, prog []byte) {
	ret, rp := vm.pc, vm.prog
	vm.pc = 0
	vm.prog = prog // for errors
	for vm.pc < len(prog) {
		step(vm)
	}
	vm.pc, vm.prog = ret, rp
}

func step(vm *vm) {
	opcode, data, n := decodeInst(vm.prog[vm.pc:])
	vm.traceOp(vm.data, opcode, data, vm.prog[vm.pc:])
	vm.pc += n
	if opcode == BaseData {
		vm.data.PushBytes(data)
	} else if opcode >= BaseInt {
		vm.data.PushInt64(int64(opcode) - BaseInt)
	} else {
		optab[opcode](vm)
	}
}

func decodeInst(buf []byte) (opcode byte, imm []byte, n int) {
	v, n := binary.Uvarint(buf) // note v=0 on error
	if v < BaseData {
		return byte(v), nil, n
	}
	r := v - BaseData + uint64(n)
	return BaseData, append([]byte{}, buf[n:r]...), int(r)
}

func idsEqual(a, b []byte) bool {
	if len(a) != len(b) || len(a) != 32 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func getStack(vm *vm, t int64) *tupleStack {
	switch t {
	case StackInput:
		return &vm.inputs
	case StackValue:
		return &vm.values
	case StackOutput:
		return &vm.outputs
	case StackCond:
		return &vm.conditions
	case StackNonce:
		return &vm.nonces
	case StackAnchor:
		return &vm.anchors
	case StackRetirement:
		return &vm.retirements
	case StackTimeConstraint:
		return &vm.timeconstraints
	case StackAnnotation:
		return &vm.annotations
	case StackSummary:
		return &vm.summary
	default:
		panic(errors.New("bad stack identifier"))
	}
}
