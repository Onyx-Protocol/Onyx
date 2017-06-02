package txvm

import (
	"encoding/binary"
	"fmt"

	"github.com/chain/txvm/data"
	"github.com/chain/txvm/op"

	"chain/errors"
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
	Version     int64
	Tmin, Tmax  int64
	Runlimit    int64
	In, Nonce   []ID
	Out, Retire []ID
	Data        ID
	ExtHash     ID
	Proof       []byte
}

type vm struct {
	// config, doesn't change after init
	traceUnlock func(Contract)
	traceLock   func(Contract)
	traceOp     func(data.List, []byte)
	tmin        int64
	tmax        int64

	pc   int    // program counter
	prog []byte // current program

	data data.List

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

func Validate(x *Tx, o ...Option) (err error) {
	vm := &vm{
		input:       x.In,
		traceUnlock: func(Contract) {},
		traceLock:   func(Contract) {},
	}
	for _, o := range o {
		o(vm)
	}

	defer func() {
		if x := recover(); x != nil {
			err = errors.Wrap(&ExecError{vm.pc, vm.prog, x.(error)})
		}
	}()

	exec(vm, x.Proof)

	switch {
	case len(vm.input) > 0:
		return errors.New("unused inputs")
	case len(vm.value) > 0:
		return errors.New("unused values")
	case len(vm.pred) > 0:
		return errors.New("unused predicates")
	case len(vm.contract) > 0:
		return errors.New("unused contracts")
	case !idsEqual(vm.output, x.Out):
		return errors.New("output mismatch")
	case !idsEqual(vm.nonce, x.Nonce):
		return errors.New("nonce mismatch")
	}
	return nil
}

func exec(vm *vm, prog []byte) {
	ret, rp := vm.pc, vm.prog
	vm.pc = 0
	vm.prog = prog // for errors
	for vm.pc < len(prog) {
		vm.traceOp(vm.data, prog[vm.pc:])
		opcode, data, n := decodeInst(prog[vm.pc:])
		vm.pc += n
		if opcode == op.BaseData {
			vm.data.PushBytes(data)
		} else if opcode >= op.MinInt {
			vm.data.PushInt64(int64(opcode) - op.BaseInt)
		} else {
			optab[opcode](vm)
		}
	}
	vm.pc, vm.prog = ret, rp
}

func decodeInst(buf []byte) (opcode byte, imm []byte, n int) {
	v, n := binary.Uvarint(buf) // note v=0 on error
	if v < op.BaseData {
		return byte(v), nil, n
	}
	r := v - op.BaseData + uint64(n)
	return op.BaseData, buf[n:r], int(r)
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

type ExecError struct {
	PC   int
	Prog []byte
	Err  error
}

func (e ExecError) Error() string {
	return fmt.Sprintf("pc 0x%x: %s", e.PC, e.Err)
}
