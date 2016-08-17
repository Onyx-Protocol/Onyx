package vm

import "encoding/binary"

func opFalse(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.pushBool(false, false)
}

func mkOpData(n int) func(*virtualMachine) error {
	return func(vm *virtualMachine) error {
		if vm.pc+1+uint32(n) > uint32(len(vm.program)) {
			return ErrShortProgram
		}
		err := vm.applyCost(1)
		if err != nil {
			return err
		}
		d := make([]byte, n)
		copy(d, vm.program[vm.pc+1:vm.pc+1+uint32(n)])
		return vm.push(d, false)
	}
}

func opPushdata(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if vm.pc >= uint32(len(vm.program)) {
		return ErrShortProgram
	}
	n, nbytes := binary.Uvarint(vm.program[vm.pc+1:])
	if nbytes <= 0 {
		return ErrBadValue
	}
	start := uint64(vm.pc) + 1 + uint64(nbytes)
	end := start + uint64(n)
	if end > uint64(len(vm.program)) {
		return ErrShortProgram
	}
	d := make([]byte, end-start)
	copy(d, vm.program[start:end])
	return vm.push(d, false)
}

func op1Negate(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.pushInt64(-1, false)
}

func mkOpNum(n uint8) func(*virtualMachine) error {
	return func(vm *virtualMachine) error {
		err := vm.applyCost(1)
		if err != nil {
			return err
		}
		return vm.pushInt64(int64(n), false)
	}
}

func opNop(_ *virtualMachine) error {
	return nil
}

func pushdataBytes(in []byte) []byte {
	l := len(in)
	if l == 0 {
		return []byte{OP_0}
	}
	if l <= 75 {
		return append([]byte{OP_DATA_1 + uint8(len(in)) - 1}, in...)
	}
	var lenBytes [10]byte
	nbytes := binary.PutUvarint(lenBytes[:], uint64(l))
	res := append([]byte{OP_PUSHDATA}, lenBytes[:nbytes]...)
	res = append(res, in...)
	return res
}

func pushdataInt64(n int64) []byte {
	if n == 0 {
		return []byte{OP_0}
	}
	if n >= 1 && n <= 16 {
		return []byte{OP_1 + uint8(n) - 1}
	}
	return pushdataBytes(Int64Bytes(n))
}
