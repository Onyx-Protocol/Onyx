package vm

import (
	"crypto/sha256"
	"hash"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/math/checked"
)

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
	return vm.pushBool(ed25519.Verify(ed25519.PublicKey(pubkeyBytes), msg, sig), true)
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
	sigs := make([][]byte, 0, numSigs)
	for i := int64(0); i < numSigs; i++ {
		sig, err := vm.pop(true)
		if err != nil {
			return err
		}
		sigs = append(sigs, sig)
	}

	pubkeys := make([]ed25519.PublicKey, 0, numPubkeys)
	for _, p := range pubkeyByteses {
		if len(p) != ed25519.PublicKeySize {
			return vm.pushBool(false, true)
		}
		pubkeys = append(pubkeys, ed25519.PublicKey(p))
	}

	for len(sigs) > 0 && len(pubkeys) > 0 {
		if ed25519.Verify(pubkeys[0], msg, sigs[0]) {
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
	h := vm.txContext.TxSigHash
	return vm.push(h[:], false)
}

func opBlockHash(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	h := vm.block.Hash()
	err := vm.applyCost(4 * int64(len(h)))
	if err != nil {
		return err
	}
	return vm.push(h[:], false)
}
