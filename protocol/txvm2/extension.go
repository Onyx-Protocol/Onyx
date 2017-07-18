package txvm2

func nop(vm *vm) {
	if vm.extension {
		panic(ErrFail)
	}
}

var (
	opNop0 = nop
	opNop1 = nop
	opNop2 = nop
	opNop3 = nop
	opNop4 = nop
	opNop5 = nop
	opNop6 = nop
	opNop7 = nop
	opNop8 = nop
	opNop9 = nop
)

func opReserved(vm *vm) {
	panic(ErrFail)
}
