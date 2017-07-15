package txvm2

import "bytes"

func opCreate(vm *vm) {
	data := vm.pop()
	cmd, ok := vm.stacks[commandstack].top()
	if !ok {
		panic(xxx)
	}
	rec := mkRecord(cmd[1], data)
	vm.stacks[entrystack].pushTuple(rec)
}

func opDelete(vm *vm) {
	rec := vm.stacks[recordstack].popRecord()
	cmd, ok := vm.stacks[commandstack].top()
	if !ok {
		panic(xxx)
	}
	if !bytes.Equal(rec[1], cmd.(tuple)[1]) {
		panic(xxx)
	}
}

func opComplete(vm *vm) {
	rec := vm.stacks[recordstack].popRecord()
	cmd, ok := vm.stacks[commandstack].top()
	if !ok {
		panic(xxx)
	}
	if !bytes.Equal(rec[1], cmd.(tuple)[1]) {
		panic(xxx)
	}
	vm.stacks[effectstack].pushTuple(rec)
}
