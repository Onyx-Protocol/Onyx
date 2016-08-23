package vm

import (
	"fmt"
	"io"

	// TODO(bobg): very little of this package depends on bc, consider trying to remove the dependency
	"chain/cos/bc"
)

const initialRunLimit = 50000

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
	controlStack []controlTuple
	dataStack    [][]byte
	altStack     [][]byte

	tx         *bc.Tx
	inputIndex uint32
	sigHasher  *bc.SigHasher

	block *bc.Block
}

// Set this to a non-nil value to produce trace output during
// execution.
var TraceOut io.Writer

func VerifyTxInput(tx *bc.Tx, inputIndex uint32) (bool, error) {
	txinput := tx.Inputs[inputIndex]

	var program []byte
	switch c := txinput.InputCommitment.(type) {
	case *bc.IssuanceInputCommitment:
		if c.VMVersion != 1 {
			return false, ErrUnsupportedVM
		}
		program = c.IssuanceProgram
	case *bc.SpendInputCommitment:
		if c.VMVersion != 1 {
			return false, ErrUnsupportedVM
		}
		program = c.ControlProgram
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

	for _, arg := range txinput.InputWitness {
		err := vm.push(arg, false)
		if err != nil {
			return false, err
		}
	}

	return vm.run()
}

func VerifyBlockHeader(block, prevBlock *bc.Block) (bool, error) {
	vm := virtualMachine{
		block: block,

		program:  prevBlock.ConsensusProgram,
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
	for vm.pc = 0; vm.pc < uint32(len(vm.program)); { // handle vm.pc updates in the loop
		inst, err := ParseOp(vm.program, vm.pc)
		if err != nil {
			return false, err
		}

		vm.nextPC = vm.pc + inst.Len

		var skip bool
		switch inst.Op {
		case OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF, OP_WHILE, OP_ENDWHILE:
			skip = false
		default:
			skip = len(vm.controlStack) > 0 && !vm.controlStack[len(vm.controlStack)-1].flag
		}

		if TraceOut != nil {
			opname := inst.Op.String()
			if skip {
				opname = fmt.Sprintf("[%s]", opname)
			}
			fmt.Fprintf(TraceOut, "vm %d pc %d limit %d %s", vm.depth, vm.pc, vm.runLimit, opname)
			if len(inst.Data) > 0 {
				fmt.Fprintf(TraceOut, " %x", inst.Data)
			}
			fmt.Fprint(TraceOut, "\n")
		}

		if !skip {
			vm.deferredCost = 0
			vm.data = inst.Data
			err := ops[inst.Op].fn(vm)
			if err != nil {
				return false, err
			}
			err = vm.applyCost(vm.deferredCost)
			if err != nil {
				return false, err
			}
		} else {
			vm.applyCost(1)
		}

		vm.pc = vm.nextPC

		if TraceOut != nil && !skip {
			for i := len(vm.dataStack) - 1; i >= 0; i-- {
				fmt.Fprintf(TraceOut, "  stack %d: %x\n", len(vm.dataStack)-1-i, vm.dataStack[i])
			}
		}
	}

	if len(vm.controlStack) > 0 {
		return false, ErrNonEmptyControlStack
	}

	res := len(vm.dataStack) > 0 && AsBool(vm.dataStack[len(vm.dataStack)-1])
	return res, nil
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
