package vm

import (
	"fmt"
	"io"

	// TODO(bobg): very little of this package depends on bc, consider trying to remove the dependency
	"chain/protocol/bc"
)

const initialRunLimit = 10000

type virtualMachine struct {
	program      []byte
	pc, nextPC   uint32
	runLimit     int64
	deferredCost int64

	// Stores the data parsed out of an opcode. Used as input to
	// data-pushing opcodes.
	data []byte

	// CHECKPREDICATE spawns a child vm with depth+1
	depth int

	// In each of these stacks, stack[len(stack)-1] is the top element.
	dataStack [][]byte
	altStack  [][]byte

	tx         *bc.Tx
	inputIndex int
	sigHasher  *bc.SigHasher

	block *bc.Block
}

// TraceOut - if non-nil - will receive trace output during
// execution.
var TraceOut io.Writer

func VerifyTxInput(tx *bc.Tx, inputIndex int) (ok bool, err error) {
	defer func() {
		if panErr := recover(); panErr != nil {
			ok = false
			err = ErrUnexpected
		}
	}()
	return verifyTxInput(tx, inputIndex)
}

func verifyTxInput(tx *bc.Tx, inputIndex int) (bool, error) {
	if inputIndex < 0 || inputIndex >= len(tx.Inputs) {
		return false, ErrBadValue
	}

	txinput := tx.Inputs[inputIndex]

	var program []byte
	switch inp := txinput.TypedInput.(type) {
	case *bc.IssuanceInput:
		if inp.VMVersion != 1 {
			return false, ErrUnsupportedVM
		}
		program = inp.IssuanceProgram
	case *bc.SpendInput:
		if inp.VMVersion != 1 {
			return false, ErrUnsupportedVM
		}
		program = inp.ControlProgram
	default:
		return false, ErrUnsupportedTx
	}

	vm := virtualMachine{
		tx:         tx,
		inputIndex: inputIndex,
		sigHasher:  bc.NewSigHasher(&tx.TxData),

		program:  program,
		runLimit: initialRunLimit,
	}

	for _, arg := range txinput.Arguments() {
		err := vm.push(arg, false)
		if err != nil {
			return false, err
		}
	}

	return vm.run()
}

func VerifyBlockHeader(prev *bc.BlockHeader, block *bc.Block) (ok bool, err error) {
	defer func() {
		if panErr := recover(); panErr != nil {
			ok = false
			err = ErrUnexpected
		}
	}()
	return verifyBlockHeader(prev, block)
}

func verifyBlockHeader(prev *bc.BlockHeader, block *bc.Block) (bool, error) {
	vm := virtualMachine{
		block: block,

		program:  prev.ConsensusProgram,
		runLimit: initialRunLimit,
	}

	for _, arg := range block.Witness {
		err := vm.push(arg, false)
		if err != nil {
			return false, err
		}
	}

	return vm.run()
}

func (vm *virtualMachine) run() (bool, error) {
	for vm.pc = 0; vm.pc < uint32(len(vm.program)); { // handle vm.pc updates in step
		err := vm.step()
		if err != nil {
			return false, err
		}
	}

	res := len(vm.dataStack) > 0 && AsBool(vm.dataStack[len(vm.dataStack)-1])
	return res, nil
}

func (vm *virtualMachine) step() error {
	inst, err := ParseOp(vm.program, vm.pc)
	if err != nil {
		return err
	}

	if vm.isDisallowedOpcode(inst.Op) {
		return ErrDisallowedOpcode
	}

	vm.nextPC = vm.pc + inst.Len

	if TraceOut != nil {
		opname := inst.Op.String()
		fmt.Fprintf(TraceOut, "vm %d pc %d limit %d %s", vm.depth, vm.pc, vm.runLimit, opname)
		if len(inst.Data) > 0 {
			fmt.Fprintf(TraceOut, " %x", inst.Data)
		}
		fmt.Fprint(TraceOut, "\n")
	}

	vm.deferredCost = 0
	vm.data = inst.Data
	err = ops[inst.Op].fn(vm)
	if err != nil {
		return err
	}
	err = vm.applyCost(vm.deferredCost)
	if err != nil {
		return err
	}

	vm.pc = vm.nextPC

	if TraceOut != nil {
		for i := len(vm.dataStack) - 1; i >= 0; i-- {
			fmt.Fprintf(TraceOut, "  stack %d: %x\n", len(vm.dataStack)-1-i, vm.dataStack[i])
		}
	}

	return nil
}

func (vm *virtualMachine) isDisallowedOpcode(op Op) bool {
	if vm.tx == nil {
		return false
	}
	if vm.tx.Version != 1 {
		return false
	}
	return isExpansion[op]
}

func (vm *virtualMachine) push(data []byte, deferred bool) error {
	cost := 8 + int64(len(data))
	if deferred {
		vm.deferCost(cost)
	} else {
		err := vm.applyCost(cost)
		if err != nil {
			return err
		}
	}
	vm.dataStack = append(vm.dataStack, data)
	return nil
}

func (vm *virtualMachine) pushBool(b bool, deferred bool) error {
	return vm.push(BoolBytes(b), deferred)
}

func (vm *virtualMachine) pushInt64(n int64, deferred bool) error {
	return vm.push(Int64Bytes(n), deferred)
}

func (vm *virtualMachine) pop(deferred bool) ([]byte, error) {
	if len(vm.dataStack) == 0 {
		return nil, ErrDataStackUnderflow
	}
	res := vm.dataStack[len(vm.dataStack)-1]
	vm.dataStack = vm.dataStack[:len(vm.dataStack)-1]

	cost := 8 + int64(len(res))
	if deferred {
		vm.deferCost(-cost)
	} else {
		vm.runLimit += cost
	}

	return res, nil
}

func (vm *virtualMachine) popInt64(deferred bool) (int64, error) {
	bytes, err := vm.pop(deferred)
	if err != nil {
		return 0, err
	}
	n, err := AsInt64(bytes)
	return n, err
}

func (vm *virtualMachine) top() ([]byte, error) {
	if len(vm.dataStack) == 0 {
		return nil, ErrDataStackUnderflow
	}
	return vm.dataStack[len(vm.dataStack)-1], nil
}

// positive cost decreases runlimit, negative cost increases it
func (vm *virtualMachine) applyCost(n int64) error {
	if n > vm.runLimit {
		return ErrRunLimitExceeded
	}
	vm.runLimit -= n
	return nil
}

func (vm *virtualMachine) deferCost(n int64) {
	vm.deferredCost += n
}

func stackCost(stack [][]byte) int64 {
	result := int64(8 * len(stack))
	for _, item := range stack {
		result += int64(len(item))
	}
	return result
}
