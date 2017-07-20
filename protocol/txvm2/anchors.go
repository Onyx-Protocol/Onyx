package txvm2

import "chain/crypto/sha3pool"

func opNonce(vm *vm) {
	min := vm.popInt64(datastack)
	max := vm.popInt64(datastack)
	bcID := vm.popBytes(datastack)
	p := vm.peekProgram(commandstack)
	nonce := nonce{p.program, min, max, bcID}
	vm.pushNonce(effectstack, nonce)
	vm.pushAnchor(entrystack, anchor{nonce.id()})
	vm.pushMintime(effectstack, mintime{min})
	vm.pushMaxtime(effectstack, maxtime{max})
}

func opReanchor(vm *vm) {
	a := vm.popAnchor(entrystack)
	vm.pushAnchor(entrystack, anchor{a.id()})
}

func opSplitAnchor(vm *vm) {
	a := vm.popAnchor(entrystack)
	id := a.id()
	var h [32]byte
	sha3pool.Sum256(h[:], append([]byte{0x01}, id...))
	vm.pushAnchor(entrystack, anchor{h[:]})
	sha3pool.Sum256(h[:], append([]byte{0x00}, id...))
	vm.pushAnchor(entrystack, anchor{h[:]})
}

func opAnchorTransaction(vm *vm) {
	a := vm.popAnchor(entrystack)
	vm.pushAnchor(effectstack, a)
}
