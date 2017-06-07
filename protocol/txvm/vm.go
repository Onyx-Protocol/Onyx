package txvm

import (
	"encoding/binary"
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
	In, Nonce        []ID
	Out, Retire      []ID
	Data             ID
	ExtHash          ID
	Proof            []byte
}

type vm struct {
	// config, doesn't change after init
	traceUnlock func(Contract)
	traceLock   func(Contract)
	traceOp     func(stack, []byte)
	tmin        int64
	tmax        int64

	pc   int    // program counter
	prog []byte // current program

	data stack

	// linear types
	input    []ID     // must end empty
	value    []*value // must end empty
	pred     []pval   // must end empty
	contract []*cval  // must end empty
	anchor   []ID

	// results
	output []ID
	nonce  []ID
	retire []ID
}

// Validate returns whether x is valid.
//
// To get detailed information about a Tx,
// such as determining why an invalid Tx is invalid,
// use Option funcs to trace execution.
func Validate(x *Tx, o ...Option) bool {
	vm := &vm{
		input:       x.In,
		traceUnlock: func(Contract) {},
		traceLock:   func(Contract) {},
	}
	for _, o := range o {
		o(vm)
	}

	defer func() { recover() }()

	exec(vm, x.Proof)

	// TODO(kr): call some tracing hook here
	// to signal end of execution.

	return len(vm.input) == 0 &&
		len(vm.value) == 0 &&
		len(vm.pred) == 0 &&
		len(vm.contract) == 0 &&
		idsEqual(vm.output, x.Out) &&
		idsEqual(vm.nonce, x.Nonce)
}

func exec(vm *vm, prog []byte) {
	ret, rp := vm.pc, vm.prog
	vm.pc = 0
	vm.prog = prog // for errors
	for vm.pc < len(prog) {
		vm.traceOp(vm.data, prog[vm.pc:])
		opcode, data, n := decodeInst(prog[vm.pc:])
		vm.pc += n
		if opcode == BaseData {
			vm.data.PushBytes(data)
		} else if opcode >= MinInt {
			vm.data.PushInt64(int64(opcode) - BaseInt)
		} else {
			optab[opcode](vm)
		}
	}
	vm.pc, vm.prog = ret, rp
}

func decodeInst(buf []byte) (opcode byte, imm []byte, n int) {
	v, n := binary.Uvarint(buf) // note v=0 on error
	if v < BaseData {
		return byte(v), nil, n
	}
	r := v - BaseData + uint64(n)
	return BaseData, buf[n:r], int(r)
}

func idsEqual(a, b []ID) bool {
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
