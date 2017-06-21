package txvm

type Option func(*vm)

type Contract struct {
	Prog   []byte
	Asset  [32]byte
	Amount uint64
}

func TraceUnlock(f func(Contract)) Option {
	return func(vm *vm) {
		vm.traceUnlock = f
	}
}

func TraceLock(f func(Contract)) Option {
	return func(vm *vm) {
		vm.traceLock = f
	}
}

func TraceOp(f func(s stack, op byte, data, prog []byte)) Option {
	// TODO(kr): provide other state too (if necessary?)
	return func(vm *vm) {
		vm.traceOp = f
	}
}

func TraceError(f func(err error)) Option {
	return func(vm *vm) {
		vm.traceError = f
	}
}
