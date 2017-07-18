package txvm2

func opBefore(vm *vm) {
	max := vm.popInt64(datastack)
	vm.push(effectstack, mkMaxtime(max))
}

func opAfter(vm *vm) {
	min := vm.popInt64(datastack)
	vm.push(effectstack, mkMintime(min))
}
