package txvm

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"

	"chain/protocol/txvm/data"
	"chain/protocol/txvm/op"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/math/checked"
)

// avoid initialization loop
func init() { optab = ops }

var optab [op.NumOp]func(*vm)
var ops = [op.NumOp]func(*vm){
	op.Fail:   func(vm *vm) { panic(errors.New("illegal instruction")) },
	op.Jump:   opJump,
	op.JumpIf: opJumpIf,
	op.Exec:   opExec,
	op.Roll:   opRoll,
	op.Depth:  opDepth,
	op.Drop:   opDrop,
	op.Dup:    opDup,

	op.Varint: opVarint,

	op.Abs:    opAbs,
	op.Add:    intBinOp(checked.AddInt64).run,
	op.Mul:    intBinOp(checked.MulInt64).run,
	op.Div:    intBinOp(checked.DivInt64).run,
	op.Mod:    intBinOp(checked.ModInt64).run,
	op.Lshift: intBinOp(checked.LshiftInt64).run,
	op.Rshift: intBinOp(rshift).run,
	op.Min:    intBinOp(min).run,
	op.Max:    intBinOp(max).run,

	op.Not:   opNot,
	op.And:   boolBinOp(func(x, y int64) bool { return x != 0 && y != 0 }),
	op.Or:    boolBinOp(func(x, y int64) bool { return x != 0 || y != 0 }),
	op.GT:    boolBinOp(func(x, y int64) bool { return x > y }),
	op.GE:    boolBinOp(func(x, y int64) bool { return x >= y }),
	op.Equal: opEqual,

	op.Cat:    opCat,
	op.Slice:  opSlice,
	op.Len:    opLen,
	op.BitNot: opBitNot,
	op.BitAnd: bitBinOp(func(x, y int64) int64 { return x & y }).run,
	op.BitOr:  bitBinOp(func(x, y int64) int64 { return x | y }).run,
	op.BitXor: bitBinOp(func(x, y int64) int64 { return x ^ y }).run,

	op.SHA256:        hashOp(sha256.New).run,
	op.SHA3:          hashOp(sha3.New256).run,
	op.CheckSig:      opCheckSig,
	op.CheckMultiSig: opCheckMultiSig,
}

func opJump(vm *vm) {
	p := vm.data.PopInt64()
	vm.pc = int(p)
}

func opJumpIf(vm *vm) {
	p := vm.data.PopInt64()
	x := vm.data.PopInt64()
	if x != 0 {
		vm.pc = int(p)
	}
}

func opExec(vm *vm) {
	prog := vm.data.PopBytes()
	exec(vm, prog)
}

func opRoll(vm *vm) {
	t := vm.data.PopInt64()
	n := vm.data.PopInt64()
	switch t {
	case op.StackData:
		vm.data.Roll(n)
	case op.StackInput:
		panic(errors.New("todo"))
	case op.StackValue:
		panic(errors.New("todo"))
	case op.StackCond:
		panic(errors.New("todo"))
	case op.StackOutput:
		panic(errors.New("todo"))
	case op.StackNonce:
		panic(errors.New("todo"))
	default:
		panic(errors.New("bad stack selector"))
	}
}

func opDepth(vm *vm) {
	t := vm.data.PopInt64()
	var n int
	switch t {
	case op.StackData:
		n = int(vm.data.Len())
	case op.StackInput:
		n = len(vm.input)
	case op.StackValue:
		n = len(vm.value)
	case op.StackCond:
		n = len(vm.pred)
	case op.StackAnchor:
		n = len(vm.anchor)
	case op.StackOutput:
		n = len(vm.output)
	case op.StackNonce:
		n = len(vm.nonce)
	case op.StackRetire:
		n = len(vm.retire)
	default:
		panic(errors.New("bad stack selector"))
	}
	vm.data.PushInt64(int64(n))
}

func opDrop(vm *vm) {
	vm.data.Pop()
}

func opDup(vm *vm) {
	x := vm.data.Pop()
	vm.data.Push(x)
	vm.data.Push(x)
}

func opAbs(vm *vm) {
	x := vm.data.PopInt64()
	if x == -x {
		panic(errors.New("range"))
	}
	if x < 0 {
		x = -x
	}
	vm.data.PushInt64(x)
}

type intBinOp func(x, y int64) (int64, bool)

func (o intBinOp) run(vm *vm) {
	y := vm.data.PopInt64()
	x := vm.data.PopInt64()
	z, ok := o(x, y)
	if !ok {
		panic(errors.New("range"))
	}
	vm.data.PushInt64(z)
}

func rshift(x, y int64) (int64, bool) {
	if y < 0 {
		return 0, false
	}
	return x >> uint64(y), true
}

func min(x, y int64) (int64, bool) {
	if x < y {
		return x, true
	}
	return y, true
}

func max(x, y int64) (int64, bool) {
	if x > y {
		return x, true
	}
	return y, true
}

func opNot(vm *vm) {
	var v bool
	switch x := vm.data.Pop().(type) {
	case data.Int64:
		v = x != 0
	default:
		v = true
	}
	vm.data.Push(data.Bool(v))
}

func boolBinOp(f func(x, y int64) bool) func(vm *vm) {
	return func(vm *vm) {
		y := vm.data.PopInt64()
		x := vm.data.PopInt64()
		vm.data.Push(data.Bool(f(x, y)))
	}
}

func opCat(vm *vm) {
	y := vm.data.PopBytes()
	x := vm.data.PopBytes()
	b := append(x[:len(x):len(x)], y...)
	vm.data.PushBytes(b)
}

func opSlice(vm *vm) {
	b := vm.data.PopInt64()
	a := vm.data.PopInt64()
	s := vm.data.PopBytes()
	t := make([]byte, b-a)
	copy(t, s[a:b])
	vm.data.PushBytes(t)
}

func opLen(vm *vm) {
	s := vm.data.PopBytes()
	vm.data.Push(data.Int64(len(s)))
}

func opBitNot(vm *vm) {
	x := vm.data.Pop()
	switch x := x.(type) {
	case data.Int64:
		vm.data.Push(^x)
	case data.Bytes:
		z := make([]byte, len(x))
		for i := range x {
			z[i] = ^x[i]
		}
		vm.data.PushBytes(z)
	}
}

type bitBinOp func(x, y int64) int64

func (o bitBinOp) run(vm *vm) {
	y := vm.data.Pop()
	x := vm.data.Pop()
	switch x := x.(type) {
	case data.Int64:
		y := y.(data.Int64)
		vm.data.PushInt64(o(int64(x), int64(y)))
	case data.Bytes:
		y := y.(data.Bytes)
		if len(x) != len(y) {
			panic(errors.New("unequal length"))
		}
		z := make(data.Bytes, len(x))
		for i := range x {
			xi := int64(x[i])
			yi := int64(y[i])
			z[i] = byte(o(xi, yi))
		}
		vm.data.Push(z)
	}
}

type hashOp func() hash.Hash

func (o hashOp) run(vm *vm) {
	s := vm.data.PopBytes()
	h := o()
	h.Write(s)
	vm.data.Push(data.Bytes(h.Sum(nil)))
}

func opCheckSig(vm *vm) {
	k := ed25519.PublicKey(vm.data.PopBytes())
	m := vm.data.PopBytes()
	s := vm.data.PopBytes()
	if len(m) != 32 {
		panic(errors.New("message len"))
	} else if len(k) != ed25519.PublicKeySize {
		panic(errors.New("key len"))
	}
	vm.data.Push(data.Bool(ed25519.Verify(k, m, s)))
}

func opCheckMultiSig(vm *vm) {
	nkey := int64(vm.data.PopInt64())
	nsig := int64(vm.data.PopInt64())
	if nkey < 0 || nsig < 0 {
		panic(errors.New("range"))
	} else if nsig > nkey || nsig == 0 && nkey > 0 {
		panic(errors.New("bad value"))
	}

	var key []ed25519.PublicKey
	for i := int64(0); i < nkey; i++ {
		k := ed25519.PublicKey(vm.data.PopBytes())
		if len(k) != ed25519.PublicKeySize {
			panic(errors.New("key len"))
		}
		key = append(key, k)
	}

	msg := vm.data.PopBytes()
	if len(msg) != 32 {
		panic(errors.New("message len"))
	}

	var sig []data.Bytes
	for i := int64(0); i < nsig; i++ {
		sig = append(sig, vm.data.PopBytes())
	}

	for len(sig) > 0 && len(key) > 0 {
		if ed25519.Verify(key[0], msg, sig[0]) {
			sig = sig[1:]
		}
		key = key[1:]
	}
	vm.data.Push(data.Bool(len(sig) == 0))
}

func opVarint(vm *vm) {
	buf := vm.data.PopBytes()
	x, n := binary.Uvarint(buf)
	if n <= 0 {
		panic("bad varint")
	}
	vm.data.PushInt64(int64(x))
}

func opEqual(vm *vm) {
	b := vm.data.Pop()
	a := vm.data.Pop()
	var ok bool
	switch a := a.(type) {
	case data.Int64:
		b := b.(data.Int64)
		ok = a == b
	}
	vm.data.Push(data.Bool(ok))
}
