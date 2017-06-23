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
	ID:    opID,

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

	Cat:   opCat,
	Slice: opSlice,

	BitNot: opBitNot,
	BitAnd: bitBinOp(func(x, y int64) int64 { return x & y }).run,
	BitOr:  bitBinOp(func(x, y int64) int64 { return x | y }).run,
	BitXor: bitBinOp(func(x, y int64) int64 { return x ^ y }).run,

	Encode: opEncode,
	Varint: opVarint,

	MakeTuple: opMakeTuple,
	Untuple:   opUntuple,
	Field:     opField,

	Type: opType,

	SHA256:        hashOp(sha256.New).run,
	SHA3:          hashOp(sha3.New256).run,
	CheckSig:      opCheckSig,
	CheckMultiSig: opCheckMultiSig,

	Defer:        opDefer,
	Satisfy:      opSatisfy,
	Unlock:       opUnlock,
	UnlockOutput: opUnlockOutput,
	Merge:        opMerge,
	Split:        opSplit,
	Lock:         opLock,
	Retire:       opRetire,
	Anchor:       opAnchor,
	Issue:        opIssue,
	Header:       opHeader,
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

func opID(vm *vm) {
	t := vm.data.PopInt64()
	vm.data.PushBytes(getStack(vm, t).ID())
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
		for i := len(v) - 1; i >= 0; i-- {
			b = append(b, encode(v[i])...)
		}
		b = append(b, pushInt64(int64(len(v)))...)
		b = append(b, MakeTuple)
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

func opMakeTuple(vm *vm) {
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

func opDefer(vm *vm) {
	vm.conditions.Push(VMTuple{Bytes(vm.data.PopBytes())})
}

func opSatisfy(vm *vm) {
	tuple := vm.conditions.Pop()
	exec(vm, tuple[0].(Bytes))
}

func opUnlock(vm *vm) {
	input := vm.data.PopTuple()
	if !checkTuple(input, OutputTuple) {
		panic(errors.New("expected output tuple"))
	}
	vm.inputs.Push(input)
	vals := input[3].(VMTuple)
	for _, v := range vals {
		value := v.(VMTuple)
		if !checkTuple(value, ValueTuple) {
			panic(errors.New("expected value tuple"))
		}
		vm.values.Push(value)
	}
	vm.anchors.Push(VMTuple{
		Bytes(AnchorTuple),
		VMTuple{},
		historyID(Unlock, 0, input),
	})
	exec(vm, input[4].(Bytes))
}

func opUnlockOutput(vm *vm) {
	output := vm.data.PopTuple()
	if !checkTuple(output, OutputTuple) {
		panic(errors.New("expected output tuple"))
	}
	vals := output[3].(VMTuple)
	for _, v := range vals {
		value := v.(VMTuple)
		if !checkTuple(value, ValueTuple) {
			panic(errors.New("expected value tuple"))
		}
		vm.values.Push(value)
	}
	exec(vm, output[4].(Bytes))
}

func opMerge(vm *vm) {
	val1 := vm.values.Pop()
	val2 := vm.values.Pop()

	if !idsEqual(val1[4].(Bytes), val2[4].(Bytes)) {
		panic(errors.New("merging different assets"))
	}

	assetid := val1[4].(Bytes)
	sum := int64(val1[3].(Int64))
	var ok bool
	sum, ok = checked.AddInt64(sum, int64(val2[3].(Int64)))
	if !ok {
		panic(errors.New("range"))
	}

	vm.values.Push(VMTuple{
		Bytes(ValueTuple),
		VMTuple{},
		historyID(Merge, 0, val1, val2),
		Int64(sum),
		assetid,
	})
}

func opSplit(vm *vm) {
	val := vm.values.Pop()
	amt := vm.data.PopInt64()

	originalAmt := int64(val[3].(Int64))

	if amt >= originalAmt {
		panic(errors.New("split value must be less"))
	}

	vm.values.Push(VMTuple{
		Bytes(ValueTuple),
		VMTuple{},
		historyID(Split, 0, val, Int64(amt)),
		Int64(amt),
		val[4],
	})

	vm.values.Push(VMTuple{
		Bytes(ValueTuple),
		VMTuple{},
		historyID(Split, 1, val, Int64(amt)),
		Int64(originalAmt - amt),
		val[4],
	})
}

func opLock(vm *vm) {
	refData := vm.data.PopBytes()
	n := vm.data.PopInt64()
	var values VMTuple
	for i := int64(0); i < n; i++ {
		values = append(values, vm.values.Pop())
	}
	prog := vm.data.PopBytes()

	historyArgs := append(append([]Value{Bytes(refData), Int64(n)}, values...), Bytes(prog))

	vm.outputs.Push(VMTuple{
		Bytes(OutputTuple),
		VMTuple{Bytes(refData)},
		historyID(Lock, 0, historyArgs...),
		values,
		Bytes(prog),
	})
}

func opRetire(vm *vm) {
	val := vm.values.Pop()
	vm.retirements.Push(VMTuple{
		Bytes(RetirementTuple),
		VMTuple{},
		historyID(Retire, 0, val),
	})
}

func opAnchor(vm *vm) {
	tuple := vm.data.PopTuple()
	if !checkTuple(tuple, NonceTuple) {
		panic("expected nonce tuple")
	}
	vm.nonces.Push(tuple)
	vm.anchors.Push(VMTuple{
		Bytes(AnchorTuple),
		VMTuple{},
		historyID(Anchor, 0, tuple),
	})
	exec(vm, tuple[2].(Bytes))
}

func opIssue(vm *vm) {
	assetDef := vm.data.PopTuple()
	if !checkTuple(assetDef, AssetDefinitionTuple) {
		panic("expected asset definition tuple")
	}
	amount := vm.data.PopInt64()
	anchor := vm.anchors.Pop()
	assetID := calcID(assetDef)
	vm.values.Push(VMTuple{
		Bytes(ValueTuple),
		VMTuple{},
		historyID(Issue, 0, assetDef, Int64(amount), anchor),
		Int64(amount),
		Bytes(assetID),
	})
	exec(vm, assetDef[2].(Bytes))
}

func opHeader(vm *vm) {
	if vm.txheader.Len() > 0 {
		panic(errors.New("txheader already created"))
	}
	var (
		inputs      VMTuple
		outputs     VMTuple
		nonces      VMTuple
		historyArgs []Value
	)
	for vm.inputs.Len() > 0 {
		inputs = append(inputs, Bytes(vm.inputs.ID()))
		historyArgs = append(historyArgs, vm.inputs.Pop())
	}
	for vm.outputs.Len() > 0 {
		outputs = append(outputs, Bytes(vm.outputs.ID()))
		historyArgs = append(historyArgs, vm.outputs.Pop())
	}
	for vm.nonces.Len() > 0 {
		nonces = append(nonces, Bytes(vm.nonces.ID()))
		historyArgs = append(historyArgs, vm.nonces.Pop())
	}
	for vm.retirements.Len() > 0 {
		historyArgs = append(historyArgs, vm.retirements.Pop())
	}
	refData := vm.data.PopBytes()
	minTime := vm.data.PopInt64()
	maxTime := vm.data.PopInt64()

	if minTime < 0 || maxTime < minTime {
		panic(errors.New("invalid time range"))
	}

	historyArgs = append(historyArgs, Bytes(refData), Int64(minTime), Int64(maxTime))
	vm.txheader.Push(VMTuple{
		Bytes(TxHeaderTuple),
		VMTuple{Bytes(refData)},
		historyID(Header, 0, historyArgs...),
		inputs,
		outputs,
		nonces,
		Int64(minTime),
		Int64(maxTime),
	})
}

func historyID(op byte, idx int, vals ...Value) Bytes {
	history := VMTuple{
		Bytes([]byte{op}),
		append(VMTuple{}, vals...),
		Int64(idx),
	}
	return Bytes(calcID(history))
}

func checkTuple(v VMTuple, expected string) bool {
	if len(v) != len(tupleContents[expected]) {
		return false
	}
	for i := range v {
		if v[i].typ() != tupleContents[expected][i] {
			return false
		}
	}
	return true
}
