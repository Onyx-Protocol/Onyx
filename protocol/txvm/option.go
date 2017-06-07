package txvm

type Option func(*vm)

type Contract struct {
	Prog   []byte
	Asset  ID
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

func TraceOp(f func(s stack, prog []byte)) Option {
	// TODO(kr): provide other state too (if necessary?)
	return func(vm *vm) {
		vm.traceOp = f
	}
}
