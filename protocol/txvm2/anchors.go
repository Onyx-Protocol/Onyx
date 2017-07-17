package txvm2

func opNonce(vm *vm) {
	min := vm.popInt64(datastack)
	max := vm.popInt64(datastack)
	bcID := vm.popBytes(datastack)
	p := vm.peekTuple(commandstack, commandTuple)
	
}
