package txvm

type Option func(*vm)

func TraceOp(f func(s stack, op byte, data []byte)) Option {
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
