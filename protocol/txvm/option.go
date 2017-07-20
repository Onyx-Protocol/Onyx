package txvm

type Option func(*vm)

func TraceOp(f OpTracer) Option {
	return func(vm *vm) {
		vm.traceOp = f
	}
}

func TraceError(f func(err error)) Option {
	return func(vm *vm) {
		vm.traceError = f
	}
}
