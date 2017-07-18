package txvm2

import (
	"bytes"
	"errors"
)

var ErrRecord = errors.New("unauthorized record operation")

func opCreate(vm *vm) {
	data := vm.pop(datastack)
	cmd := vm.peekTuple(commandstack, programTuple)
	rec := mkRecord(programProgram(cmd), data)
	vm.push(entrystack, rec)
}

func opDelete(vm *vm) {
	rec := vm.popTuple(entrystack, recordTuple)
	cmd := vm.peekTuple(commandstack, programTuple)
	if !bytes.Equal(recordCommandProgram(rec), programProgram(cmd)) {
		panic(ErrRecord)
	}
}

func opComplete(vm *vm) {
	rec := vm.popTuple(entrystack, recordTuple)
	cmd := vm.peekTuple(commandstack, programTuple)
	if !bytes.Equal(recordCommandProgram(rec), programProgram(cmd)) {
		panic(ErrRecord)
	}
	vm.push(effectstack, rec)
}
