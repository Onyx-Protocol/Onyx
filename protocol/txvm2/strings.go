package txvm2

func opCat(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	vm.push(datastack, vbytes(append(a, b...)))
}

func opSlice(vm *vm) {
	start := vm.popInt64(datastack)
	if start < 0 {
		panic(vm.errf("slice: negative start %d", start))
	}
	end := vm.popInt64(datastack)
	if end < start {
		panic(vm.errf("slice: end %d precedes start %d", end, start))
	}
	str := vm.popBytes(datastack)
	if end > int64(len(str)) {
		panic(vm.errf("slice: end %d exceeds length %d of string %x", end, len(str), str))
	}
	vm.push(datastack, vbytes(str[start:end]))
}
