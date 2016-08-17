package vm

import (
	"encoding/binary"
	"fmt"
	"io"

	// TODO(bobg): very little of this package depends on bc, consider trying to remove the dependency
	"chain/cos/bc"
)

const initialRunLimit = 50000

type virtualMachine struct {
	program      []byte
	pc           uint32
	runLimit     int64
	deferredCost int64

	// CHECKPREDICATE spawns a child vm with depth+1
	depth int

	// In each of these stacks, stack[len(stack)-1] is the top element.
	condStack []bool
	loopStack []uint32 // each element is a pc value
	dataStack [][]byte
	altStack  [][]byte

	tx         *bc.Tx
	inputIndex uint32
	sigHasher  *bc.SigHasher

	block *bc.Block

	traceOut io.Writer
}

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

func VerifyBlock(block, prevBlock *bc.Block) (bool, error) {
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
		opcode := vm.program[vm.pc]

		switch opcode {
		case 0x65, 0x66:
			return false, ErrIllegalOpcode
		}

		op := ops[opcode]
		if op.fn == nil {
			return false, ErrUnknownOpcode
		}

		var skip bool
		switch opcode {
		case OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF, OP_WHILE, OP_ENDWHILE:
			skip = false
		default:
			skip = len(vm.condStack) > 0 && !vm.condStack[len(vm.condStack)-1]
		}

		lsDepth := len(vm.loopStack)
		var lsTop uint32
		if lsDepth > 0 {
			lsTop = vm.loopStack[len(vm.loopStack)-1]
		}

		if vm.traceOut != nil {
			opname := op.name
			if skip {
				opname = fmt.Sprintf("[%s]", opname)
			}
			fmt.Fprintf(vm.traceOut, "vm %d pc %d limit %d %s\n", vm.depth, vm.pc, vm.runLimit, opname)
		}

		if !skip {
			vm.deferredCost = 0
			err := op.fn(vm)
			if err != nil {
				return false, err
			}
			err = vm.applyCost(vm.deferredCost)
			if err != nil {
				return false, err
			}
		}

		if vm.traceOut != nil && !skip {
			for i := len(vm.dataStack) - 1; i >= 0; i-- {
				fmt.Fprintf(vm.traceOut, "  stack %d: %x\n", len(vm.dataStack)-1-i, vm.dataStack[i])
			}
		}

		switch {
		case opcode == OP_ENDWHILE:
			if len(vm.loopStack) < lsDepth {
				vm.pc = lsTop
			} else {
				vm.pc++
			}
		case opcode >= OP_DATA_1 && opcode <= OP_DATA_75:
			vm.pc += uint32(opcode - OP_DATA_1 + 2)
		case opcode == OP_PUSHDATA:
			if vm.pc >= uint32(len(vm.program)) {
				return false, ErrShortProgram
			}
			n, nbytes := binary.Uvarint(vm.program[vm.pc+1:])
			if nbytes <= 0 {
				return false, ErrBadValue
			}
			// TODO(bobg): range-check nbytes and n
			vm.pc += 1 + uint32(nbytes) + uint32(n)
		default:
			vm.pc++
		}
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
