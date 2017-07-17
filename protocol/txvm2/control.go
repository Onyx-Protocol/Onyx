package txvm2

import "errors"

var ErrFail = errors.New("fail")

func opFail(vm *vm) {
	panic(ErrFail)
}

func opPC(vm *vm) {
	vm.push(datastack, vint64(vm.run.pc))
}

func opJumpIf(vm *vm) {
	dest := vm.popInt64(datastack)
	cond := vm.popBool(datastack)
	if !cond {
		return
	}
	if dest < 0 {
		panic(xxx)
	}
	if dest > len(vm.run.prog) {
		panic(xxx)
	}
	vm.run.pc = dest
}
