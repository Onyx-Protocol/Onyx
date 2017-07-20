package txvm2

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/ecmath"
)

var ErrBadPoint = errors.New("bad ed25519 point")

func opSHA256(vm *vm) {
	a := vm.popBytes(datastack)
	h := sha256.New()
	h.Write(a)
	vm.push(datastack, vbytes(h.Sum(nil)))
}

func opSHA3(vm *vm) {
	a := vm.popBytes(datastack)
	h := sha3.New256()
	h.Write(a)
	vm.push(datastack, vbytes(h.Sum(nil)))
}

func opCheckSig(vm *vm) {
	msg := vm.popBytes(datastack)
	pubkey := vm.popBytes(datastack)
	sig := vm.popBytes(datastack)
	vm.pushBool(datastack, ed25519.Verify(ed25519.PublicKey(pubkey), msg, sig))
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
	x.SetInt64(int64(x64))
	p := popPoint(vm)
	p.ScMul(p, &x)
	res := p.Encode()
	vm.push(datastack, vbytes(res[:]))
}

func popPoint(vm *vm) *ecmath.Point {
	bBytes := vm.popBytes(datastack)
	if len(bBytes) != 32 {
		panic(ErrBadPoint)
	}
	var b [32]byte
	copy(b[:], bBytes)
	var B ecmath.Point
	_, ok := B.Decode(b)
	if !ok {
		panic(ErrBadPoint)
	}
	return &B
}
