package txvm2

import (
	"errors"
	"fmt"
)

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
		panic(fmt.Errorf("jumpif: negative destination %d", dest))
	}
	if dest > int64(len(vm.run.prog)) {
		panic(fmt.Errorf("jumpif: destination %d beyond end of %d-byte program %s", dest, len(vm.run.prog), vm.run.prog))
	}
	vm.run.pc = int64(dest)
}
