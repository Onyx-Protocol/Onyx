package txvm

import "encoding/binary"

func opEncode(vm *vm) {
	s := vm.popBytes(datastack)
	vm.push(datastack, vbytes(encode(vbytes(s))))
}

func opInt64(vm *vm) {
	a := vm.popBytes(datastack)
	res, n := binary.Varint(a)
	if n <= 0 {
		panic(vm.errf("int64: not a valid varint: %x", a))
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
	data, n, err := decodePushdata(vm.run.prog[vm.run.pc:])
	if err != nil {
		panic(vm.wraperr(err))
	}
	vm.push(datastack, vbytes(data))
	vm.run.pc += n
}
