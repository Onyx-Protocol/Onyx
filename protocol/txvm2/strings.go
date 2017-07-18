package txvm2

import "fmt"

func opCat(vm *vm) {
	a := vm.popBytes(datastack)
	b := vm.popBytes(datastack)
	vm.push(append(a, b...))
}

func opSlice(vm *vm) {
	start := vm.popInt64(datastack)
	if start < 0 {
		panic(fmt.Errorf("slice: negative start %d", start))
	}
	end := vm.popInt64(datastack)
	if end < start {
		panic(fmt.Errorf("slice: end %d precedes start %d", end, start))
	}
	str := vm.popBytes(datastack)
	if end > len(str) {
		panic(fmt.Errorf("slice: end %d exceeds length %d of string %x", end, len(str), str))
	}
	vm.push(str[start:end])
}
