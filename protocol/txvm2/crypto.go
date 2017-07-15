package txvm2

import (
	"crypto/sha256"

	"chain/crypto/ed25519"

	"golang.org/x/crypto/sha3"
)

func opSHA256(vm *vm) {
	a := vm.popString()
	h := sha256.New()
	h.Write(a)
	vm.pushString(vstring(h.Sum(nil)))
}

func opSHA3(vm *vm) {
	a := vm.popString()
	h := sha3.New256()
	h.Write(a)
	vm.pushString(vstring(h.Sum(nil)))
}

func opCheckSig(vm *vm) {
	pubkey := vm.popString()
	msg := vm.popString()
	sig := vm.popString()
	vm.pushBool(ed25519.Verify(pubkey, msg, sig))
}
