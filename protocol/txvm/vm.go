package txvm

import (
	"encoding/binary"
	"errors"
)

// Tx contains the full transaction data.
// Most of the information is contained in Proof,
// a VM program that transforms elements of In to
// elements of Out according to the constrained
// rules of the TXVM.
//
// There are some operations that transaction
// processors need to be able to do without first
// executing the proof. The other fields exist
// to facilitate those things.
// A notable example is computing the txid.
type Tx struct {
	Version          int64
	MinTime, MaxTime uint64
	Runlimit         int64
	In, Nonce        [][32]byte
	Out, Retire      [][32]byte

	Data    [32]byte
	ExtHash [32]byte
	Proof   []byte
}

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

	inputs      tupleStack
	values      tupleStack
	outputs     tupleStack
	conditions  tupleStack
	nonces      tupleStack
	anchors     tupleStack
	retirements tupleStack
	txheader    tupleStack
}

// Validate returns whether x is valid.
//
// To get detailed information about a Tx,
// such as determining why an invalid Tx is invalid,
// use Option funcs to trace execution.
func Validate(x *Tx, o ...Option) bool {
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

	exec(vm, x.Proof)

	// TODO(kr): call some tracing hook here
	// to signal end of execution.

	// check ids are equal

	return vm.conditions.Len() == 0 &&
		vm.anchors.Len() == 0
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

func idsEqual(a, b [32]byte) bool {
	if len(a) != len(b) {
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
	case StackTxHeader:
		return &vm.txheader
	default:
		panic(errors.New("bad stack identifier"))
	}
}
