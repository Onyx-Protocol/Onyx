package vm

import (
	"crypto/sha1"
	"crypto/sha256"
	"hash"

	"github.com/agl/ed25519"

	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"

	"chain/math/checked"
)

func opRipemd160(vm *virtualMachine) error {
	return doHash(vm, ripemd160.New)
}

func opSha1(vm *virtualMachine) error {
	return doHash(vm, sha1.New)
}

func opSha256(vm *virtualMachine) error {
	return doHash(vm, sha256.New)
}

func opSha3(vm *virtualMachine) error {
	return doHash(vm, sha3.New256)
}

func doHash(vm *virtualMachine, hashFactory func() hash.Hash) error {
	x, err := vm.pop(false)
	if err != nil {
		return err
	}
	cost := int64(len(x))
	if cost < 64 {
		cost = 64
	}
	err = vm.applyCost(cost)
	if err != nil {
		return err
	}
	h := hashFactory()
	_, err = h.Write(x)
	if err != nil {
		return err
	}
	return vm.push(h.Sum(nil), false)
}

func opCheckSig(vm *virtualMachine) error {
	err := vm.applyCost(1024)
	if err != nil {
		return err
	}

	pubkeyBytes, err := vm.pop(true)
	if err != nil {
		return err
	}
	if len(pubkeyBytes) != ed25519.PublicKeySize {
		return vm.pushBool(false, true)
	}
	var pubkeybuf [ed25519.PublicKeySize]byte
	copy(pubkeybuf[:], pubkeyBytes)

	msg, err := vm.pop(true)
	if err != nil {
		return err
	}
	if len(msg) != 32 {
		return ErrBadValue
	}

	sig, err := vm.pop(true)
	if err != nil {
		return err
	}
	if len(sig) != ed25519.SignatureSize {
		return vm.pushBool(false, true)
	}
	var sigbuf [ed25519.SignatureSize]byte
	copy(sigbuf[:], sig)

	return vm.pushBool(ed25519.Verify(&pubkeybuf, msg, &sigbuf), true)
}

func opCheckMultiSig(vm *virtualMachine) error {
	numPubkeys, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	pubCost, ok := checked.MulInt64(numPubkeys, 1024)
	if numPubkeys < 0 || !ok {
		return ErrBadValue
	}
	err = vm.applyCost(pubCost)
	if err != nil {
		return err
	}
	numSigs, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if numSigs < 0 || numSigs > numPubkeys || (numPubkeys > 0 && numSigs == 0) {
		return ErrBadValue
	}
	pubkeyByteses := make([][]byte, 0, numPubkeys)
	for i := int64(0); i < numPubkeys; i++ {
		pubkeyBytes, err := vm.pop(true)
		if err != nil {
			return err
		}
		pubkeyByteses = append(pubkeyByteses, pubkeyBytes)
	}
	msg, err := vm.pop(true)
	if err != nil {
		return err
	}
	if len(msg) != 32 {
		return ErrBadValue
	}
	sigs := make([][ed25519.SignatureSize]byte, numSigs)
	for i := int64(0); i < numSigs; i++ {
		sig, err := vm.pop(true)
		if err != nil {
			return err
		}
		copy(sigs[i][:], sig)
	}

	pubkeys := make([][ed25519.PublicKeySize]byte, numPubkeys)
	for i, p := range pubkeyByteses {
		if len(p) != ed25519.PublicKeySize {
			return vm.pushBool(false, true)
		}
		copy(pubkeys[i][:], p)
	}

	for len(sigs) > 0 && len(pubkeys) > 0 {
		if ed25519.Verify(&pubkeys[0], msg, &sigs[0]) {
			sigs = sigs[1:]
		}
		pubkeys = pubkeys[1:]
	}
	return vm.pushBool(len(sigs) == 0, true)
}

func opTxSigHash(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}
	err := vm.applyCost(256)
	if err != nil {
		return err
	}
	h := vm.sigHasher.Hash(vm.inputIndex)
	return vm.push(h[:], false)
}

func opBlockSigHash(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	h := vm.block.HashForSig()
	err := vm.applyCost(4 * int64(len(h)))
	if err != nil {
		return err
	}
	return vm.push(h[:], false)
}
