package txvm

import "chain/protocol/txvm/data"

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

func TraceOp(f func(stack data.List, prog []byte)) Option {
	// TODO(kr): currently it is impossible for a client
	// package to provide f because they can't import
	// our internal package ./internal/data.
	// Figure out some way to resolve that.
	// TODO(kr): provide other state too (if necessary?)
	return func(vm *vm) {
		vm.traceOp = f
	}
}
