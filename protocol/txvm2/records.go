package txvm2

import (
	"bytes"
	"errors"
)

var ErrRecord = errors.New("unauthorized record operation")

func opCreate(vm *vm) {
	data := vm.pop(datastack)
	cmd := vm.peekProgram(commandstack)
	rec := record{cmd.program, data}
	vm.pushRecord(entrystack, rec)
}

func opDelete(vm *vm) {
	rec := vm.popRecord(entrystack)
	cmd := vm.peekProgram(commandstack)
	if !bytes.Equal(rec.commandprogram, cmd.program) {
		panic(ErrRecord)
	}
}

func opComplete(vm *vm) {
	rec := vm.popRecord(entrystack)
	cmd := vm.peekProgram(commandstack)
	if !bytes.Equal(rec.commandprogram, cmd.program) {
		panic(ErrRecord)
	}
	vm.pushRecord(effectstack, rec)
}
