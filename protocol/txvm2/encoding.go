package txvm2

import (
	"encoding/binary"
	"fmt"
)

func opEncode(vm *vm) {
	s := vm.popBytes(datastack)
	vm.push(datastack, vbytes(encode(s)))
}

func opInt64(vm *vm) {
	a := vm.popBytes(datastack)
	res, n := binary.Varint(a)
	if n <= 0 {
		panic(fmt.Errorf("int64: not a valid varint: %x", a))
	}
	vm.push(datastack, vint64(res))
}

func smallInt(n vint64) func(*vm) {
	return func(vm *vm) {
		vm.push(datastack, n)
	}
}

// xxx spec is not yet written!
func opPushdata(vm *vm) {
	l, n := binary.Varint(vm.run.prog[vm.run.pc:]) // should this be uvarint?
	if n == 0 {
		panic("pushdata: unexpected end of input reading length prefix")
	}
	if n < 0 {
		panic("pushdata: length overflow")
	}
	if l < 0 {
		panic(fmt.Errorf("pushdata: negative length %d", l))
	}
	vm.run.pc += int64(n)
	if vm.run.pc+l > int64(len(vm.run.prog)) {
		panic("pushdata: unexpected end of input reading data")
	}
	vm.push(datastack, vbytes(vm.run.prog[vm.run.pc:vm.run.pc+l]))
	vm.run.pc += l
}
