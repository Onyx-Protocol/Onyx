package txvm2

import (
	"crypto/sha256"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/ecmath"
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

func opPointAdd(vm *vm) {
	a := popPoint(vm)
	b := popPoint(vm)
	a.Add(a, b)
	c := a.Encode()
	vm.push(datastack, vbytes(c[:]))
}

func opPointSub(vm *vm) {
	a := popPoint(vm)
	b := popPoint(vm)
	a.Sub(a, b)
	c := a.Encode()
	vm.push(datastack, vbytes(c[:]))
}

func opPointMul(vm *vm) {
	x64 := vm.popInt64(datastack)
	var x ecmath.Scalar
	x.SetInt64(x64)
	p := popPoint(vm)
	p.ScMul(p, x)
	res := p.Encode()
	vm.push(datastack, vbytes(res[:]))
}

func popPoint(vm *vm) *ecmath.Point {
	bBytes := vm.popBytes(datastack)
	if len(bBytes) != 32 {
		panic(xxx)
	}
	var b [32]byte
	copy(b[:], bBytes)
	var B ecmath.Point
	_, ok := B.Decode(b)
	if !ok {
		panic(xxx)
	}
	return &B
}
