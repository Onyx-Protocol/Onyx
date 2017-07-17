package txvm2

import (
	"crypto/sha256"

	"chain/crypto/ed25519"

	"golang.org/x/crypto/sha3"
)

func opSHA256(vm *vm) {
	a := vm.popBytes()
	h := sha256.New()
	h.Write(a)
	vm.push(vbytes(h.Sum(nil)))
}

func opSHA3(vm *vm) {
	a := vm.popBytes()
	h := sha3.New256()
	h.Write(a)
	vm.push(vbytes(h.Sum(nil)))
}

func opCheckSig(vm *vm) {
	pubkey := vm.popBytes()
	msg := vm.popBytes()
	sig := vm.popBytes()
	vm.pushBool(ed25519.Verify(pubkey, msg, sig))
}
