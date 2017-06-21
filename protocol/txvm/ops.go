package txvm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/math/checked"
)

// avoid initialization loop
func init() { optab = ops }

var optab [NumOp]func(*vm)
var ops = [NumOp]func(*vm){
	Fail:   func(vm *vm) { panic(errors.New("illegal instruction")) },
	PC:     opPC,
	JumpIf: opJumpIf,

	Roll:  opRoll,
	Bury:  opBury,
	Depth: opDepth,

	Len:     opLen,
	Drop:    opDrop,
	Dup:     opDup,
	ToAlt:   opToAlt,
	FromAlt: opFromAlt,

	Equal: opEqual,
	Not:   opNot,
	And:   boolBinOp(func(x, y bool) bool { return x && y }),
	Or:    boolBinOp(func(x, y bool) bool { return x || y }),

	Add:    intBinOp(checked.AddInt64).run,
	Mul:    intBinOp(checked.MulInt64).run,
	Div:    intBinOp(checked.DivInt64).run,
	Mod:    intBinOp(checked.ModInt64).run,
	Lshift: intBinOp(checked.LshiftInt64).run,
	Rshift: intBinOp(rshift).run,
	GT:     opGT,
	GE:     opGE,
	Negate: opNegate,

	Cat:   opCat,
	Slice: opSlice,

	BitNot: opBitNot,
	BitAnd: bitBinOp(func(x, y int64) int64 { return x & y }).run,
	BitOr:  bitBinOp(func(x, y int64) int64 { return x | y }).run,
	BitXor: bitBinOp(func(x, y int64) int64 { return x ^ y }).run,

	Encode: opEncode,
	Varint: opVarint,

	Tuple:   opTuple,
	Untuple: opUntuple,
	Field:   opField,

	Type: opType,

	SHA256:        hashOp(sha256.New).run,
	SHA3:          hashOp(sha3.New256).run,
	CheckSig:      opCheckSig,
	CheckMultiSig: opCheckMultiSig,

	Anchor:  opAnchor,
	Issue:   opIssue,
	Lock:    opLock,
	Satisfy: opSatisfy,
}

func opPC(vm *vm) {
	vm.data.PushInt64(int64(vm.pc))
}

func opJumpIf(vm *vm) {
	p := vm.data.PopInt64()
	x := vm.data.Pop()
	if toBool(x) {
		vm.pc = int(p)
	}
}

func opRoll(vm *vm) {
	t := vm.data.PopInt64()
	n := vm.data.PopInt64()
	switch t {
	case StackData:
		vm.data.Roll(n)
	case StackAlt:
		vm.alt.Roll(n)
	default:
		stack := getStack(vm, t)
		stack.Roll(t)
	}
}

func opBury(vm *vm) {
	t := vm.data.PopInt64()
	n := vm.data.PopInt64()
	switch t {
	case StackData:
		vm.data.Bury(n)
	case StackAlt:
		vm.alt.Bury(n)
	default:
		getStack(vm, t).Bury(n)
	}
}

func opDepth(vm *vm) {
	t := vm.data.PopInt64()
	var n int
	switch t {
	case StackData:
		n = int(vm.data.Len())
	case StackAlt:
		n = int(vm.alt.Len())
	default:
		n = int(getStack(vm, t).Len())
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

func opToAlt(vm *vm) {
	vm.alt.Push(vm.data.Pop())
}

func opFromAlt(vm *vm) {
	vm.data.Push(vm.alt.Pop())
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

func opGT(vm *vm) {
	y := vm.data.PopInt64()
	x := vm.data.PopInt64()
	vm.data.Push(Bool(x > y))
}

func opGE(vm *vm) {
	y := vm.data.PopInt64()
	x := vm.data.PopInt64()
	vm.data.Push(Bool(x >= y))
}

func opNegate(vm *vm) {
	x := vm.data.PopInt64()
	y, ok := checked.NegateInt64(x)
	if !ok {
		panic(errors.New("range"))
	}
	vm.data.PushInt64(y)
}

func opNot(vm *vm) {
	x := toBool(vm.data.Pop())
	vm.data.Push(Bool(!x))
}

func boolBinOp(f func(x, y bool) bool) func(vm *vm) {
	return func(vm *vm) {
		y := toBool(vm.data.Pop())
		x := toBool(vm.data.Pop())
		vm.data.Push(Bool(f(x, y)))
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
	vm.data.Push(Int64(len(s)))
}

func opEqual(vm *vm) {
	b := vm.data.Pop()
	a := vm.data.Pop()
	var ok bool
	switch a := a.(type) {
	case Int64:
		b := b.(Int64)
		ok = a == b
	case Bytes:
		b := b.(Bytes)
		ok = bytes.Equal(a, b)
	case VMTuple:
		panic(errors.New("can't compare tuples"))
	}
	vm.data.Push(Bool(ok))
}

func opBitNot(vm *vm) {
	x := vm.data.Pop()
	switch x := x.(type) {
	case Int64:
		vm.data.Push(^x)
	case Bytes:
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
	case Int64:
		y := y.(Int64)
		vm.data.PushInt64(o(int64(x), int64(y)))
	case Bytes:
		y := y.(Bytes)
		if len(x) != len(y) {
			panic(errors.New("unequal length"))
		}
		z := make(Bytes, len(x))
		for i := range x {
			xi := int64(x[i])
			yi := int64(y[i])
			z[i] = byte(o(xi, yi))
		}
		vm.data.Push(z)
	}
}

func opEncode(vm *vm) {
	v := vm.data.Pop()
	vm.data.PushBytes(encode(v))
}

func encode(v Value) []byte {
	switch v := v.(type) {
	case Bytes:
		return pushData(v)
	case Int64:
		return pushInt64(int64(v))
	case VMTuple:
		var b []byte
		for i := len(v); i >= 0; i-- {
			b = append(b, encode(v[i])...)
		}
		b = append(b, pushInt64(int64(len(v)))...)
		b = append(b, Tuple)
		return b
	default:
		panic(errors.New("invalid type"))
	}
}

func opVarint(vm *vm) {
	buf := vm.data.PopBytes()
	x, n := binary.Uvarint(buf)
	if n <= 0 {
		panic("bad varint")
	}
	vm.data.PushInt64(int64(x))
}

func opTuple(vm *vm) {
	len := int(vm.data.PopInt64())
	var vals []Value
	for i := 0; i < len; i++ {
		vals = append(vals, vm.data.Pop())
	}
	vm.data.PushTuple(vals)
}

func opUntuple(vm *vm) {
	tuple := vm.data.PopTuple()
	for i := len(tuple) - 1; i >= 0; i-- {
		vm.data.Push(tuple[i])
	}
}

func opField(vm *vm) {
	t := vm.data.PopInt64()
	n := vm.data.PopInt64()

	switch t {
	case StackData:
		tuple := vm.data.Peek().(VMTuple)
		vm.data.Push(tuple[n])
	case StackAlt:
		tuple := vm.alt.Peek().(VMTuple)
		vm.data.Push(tuple[n])
	case StackInput:
	default:
		stack := getStack(vm, t)
		vm.data.Push(stack.Peek()[n])
	}
}

func opType(vm *vm) {
	v := vm.data.Peek()
	vm.data.PushInt64(int64(v.typ()))
}

type hashOp func() hash.Hash

func (o hashOp) run(vm *vm) {
	s := vm.data.PopBytes()
	h := o()
	h.Write(s)
	vm.data.Push(Bytes(h.Sum(nil)))
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
	vm.data.Push(Bool(ed25519.Verify(k, m, s)))
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

	var sig []Bytes
	for i := int64(0); i < nsig; i++ {
		sig = append(sig, vm.data.PopBytes())
	}

	for len(sig) > 0 && len(key) > 0 {
		if ed25519.Verify(key[0], msg, sig[0]) {
			sig = sig[1:]
		}
		key = key[1:]
	}
	vm.data.Push(Bool(len(sig) == 0))
}

func opAnchor(vm *vm) {
	tuple := vm.data.PopTuple()
	stackNonce := vm.nonces.Pop()
	id := tupleHash(tuple)
	if !bytes.Equal(stackNonce[0].(Bytes), id) {
		panic(errors.New("bad nonce id"))
	}
	vm.anchors.Push(VMTuple{Bytes(id[:])})
	vm.conditions.Push(VMTuple{tuple[0]})
}

func opIssue(vm *vm) {
	assetDef := vm.data.PopTuple()
	amount := vm.data.PopInt64()
	anchor := vm.anchors.Pop()
	_ = anchor
	vm.conditions.Push(VMTuple{assetDef[1]})
	assetID := tupleHash(assetDef)
	vm.values.Push(VMTuple{Int64(amount), Bytes(assetID[:]), Bool(true), VMTuple{}})
}

func opLock(vm *vm) {
	refData := vm.data.PopBytes()
	n := vm.data.PopInt64()
	var values VMTuple
	for i := int64(0); i < n; i++ {
		values = append(values, vm.values.Pop())
	}
	prog := vm.data.PopBytes()
	vm.outputs.Push(VMTuple{Bytes(prog), values, VMTuple{}, Bytes(refData)})
}

func opSatisfy(vm *vm) {
	tuple := vm.conditions.Pop()
	exec(vm, tuple[0].(Bytes))
}

func tupleHash(tuple VMTuple) []byte {
	flattened := flatten(tuple)
	hash := [32]byte{}
	sha3.TupleHash256(flattened, nil, hash[:])
	return hash[:]
}

func flatten(tuple VMTuple) [][]byte {
	var byteTuple [][]byte
	for _, v := range tuple {
		switch v := v.(type) {
		case Int64:
			buf := make([]byte, 10)
			str := append([]byte{TypeInt64}, buf[:binary.PutUvarint(buf, uint64(v))]...)
			byteTuple = append(byteTuple, str)
		case Bytes:
			byteTuple = append(byteTuple, append([]byte{TypeString}, v...))
		case VMTuple:
			byteTuple = append(byteTuple, append([]byte{TypeTuple}, tupleHash(v)...))
		}
	}
	return byteTuple
}
