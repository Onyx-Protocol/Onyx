package txvm2

import (
	"bytes"
	"fmt"

	"chain/crypto/sha3pool"
)

func opNonce(vm *vm) {
	min := vm.popInt64(datastack)
	max := vm.popInt64(datastack)
	bcID := vm.popBytes(datastack)
	p := vm.peekTuple(commandstack, programTuple)
	nonce := mkNonce(programProgram(p), min, max, bcID)
	vm.push(effectstack, nonce)
	vm.push(entrystack, mkAnchor(getID(nonce)))
	vm.push(effectstack, mkMintime(min))
	vm.push(effectstack, mkMaxtime(max))
}

func opReanchor(vm *vm) {
	a := vm.popTuple(entrystack, anchorTuple)
	vm.push(entrystack, mkAnchor(getID(a)))
}

func opSplitAnchor(vm *vm) {
	a := vm.popTuple(entrystack, anchorTuple)
	id := getID(a)
	var h [32]byte
	sha3pool.Sum256(h[:], append([]byte{0x01}, id...))
	vm.push(entrystack, mkAnchor(h[:]))
	sha3pool.Sum256(h[:], append([]byte{0x00}, id...))
	vm.push(entrystack, mkAnchor(h[:]))
}

func opAnchorTransaction(vm *vm) {
	a := vm.popTuple(entrystack, anchorTuple)
	vm.push(effectstack, a)
}
