package txvm2

func opBefore(vm *vm) {
	max := vm.popInt64(datastack)
	vm.pushMaxtime(effectstack, &maxtime{max})
}

func opAfter(vm *vm) {
	min := vm.popInt64(datastack)
	vm.pushMintime(effectstack, &mintime{min})
}
