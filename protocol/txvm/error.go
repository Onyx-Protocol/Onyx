package txvm

import (
	"errors"
	"fmt"
)

type vmerror struct {
	e  error
	vm *vm
}

func vmerr(vm *vm, msg string) vmerror {
	return vmerror{e: errors.New(msg), vm: vm}
}

func vmerrf(vm *vm, msg string, arg ...interface{}) vmerror {
	return vmerror{e: fmt.Errorf(msg, arg...), vm: vm}
}

func (v vmerror) Error() string {
	// TODO: include all kinds of stuff from v.vm
	return v.e.Error()
}
