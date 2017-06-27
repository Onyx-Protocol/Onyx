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

	Roll:    opRoll,
	Bury:    opBury,
	Reverse: opReverse,
	Depth:   opDepth,
	ID:      opID,

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

	Annotate:     opAnnotate,
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
	Before:       opBefore,
	After:        opAfter,
	Summarize:    opSummarize,
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

func opReverse(vm *vm) {
	t := vm.data.PopInt64()
	n := vm.data.PopInt64()
	switch t {
	case StackData:
		vm.data.Reverse(n)
	case StackAlt:
		vm.alt.Reverse(n)
	default:
		getStack(vm, t).Reverse(n)
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
	case Tuple:
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
	case Tuple:
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
		panic(errors.New("bad varint"))
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
		tuple := vm.data.Peek().(Tuple)
		vm.data.Push(tuple[n])
	case StackAlt:
		tuple := vm.alt.Peek().(Tuple)
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

func opAnnotate(vm *vm) {
	vm.tupleStacks[StackAnnotation].Push(Tuple{Bytes(AnnotationTuple), Bytes(vm.data.PopBytes())})
}

func opDefer(vm *vm) {
	vm.tupleStacks[StackCond].Push(Tuple{Bytes(vm.data.PopBytes())})
}

func opSatisfy(vm *vm) {
	tuple := vm.tupleStacks[StackCond].Pop()
	exec(vm, tuple[0].(Bytes))
}

func opUnlock(vm *vm) {
	input := vm.data.PopTuple()
	if !checkTuple(input, OutputTuple) {
		panic(errors.New("expected output tuple"))
	}
	vm.tupleStacks[StackInput].Push(input)
	vals := input[2].(Tuple)
	for _, v := range vals {
		value := v.(Tuple)
		if !checkTuple(value, ValueTuple) {
			panic(errors.New("expected value tuple"))
		}
		vm.tupleStacks[StackValue].Push(value)
	}
	vm.tupleStacks[StackAnchor].Push(Tuple{
		Bytes(AnchorTuple),
		historyID(Unlock, 0, input),
	})
	exec(vm, input[3].(Bytes))
}

func opUnlockOutput(vm *vm) {
	output := vm.data.PopTuple()
	if !checkTuple(output, OutputTuple) {
		panic(errors.New("expected output tuple"))
	}
	vals := output[2].(Tuple)
	for _, v := range vals {
		value := v.(Tuple)
		if !checkTuple(value, ValueTuple) {
			panic(errors.New("expected value tuple"))
		}
		vm.tupleStacks[StackValue].Push(value)
	}
	exec(vm, output[3].(Bytes))
}

func opMerge(vm *vm) {
	val1 := vm.tupleStacks[StackValue].Pop()
	val2 := vm.tupleStacks[StackValue].Pop()

	if !idsEqual(val1[3].(Bytes), val2[3].(Bytes)) {
		panic(errors.New("merging different assets"))
	}

	assetid := val1[3].(Bytes)
	sum := int64(val1[2].(Int64))
	var ok bool
	sum, ok = checked.AddInt64(sum, int64(val2[2].(Int64)))
	if !ok {
		panic(errors.New("range"))
	}

	vm.tupleStacks[StackValue].Push(Tuple{
		Bytes(ValueTuple),
		historyID(Merge, 0, val1, val2),
		Int64(sum),
		assetid,
	})
}

func opSplit(vm *vm) {
	val := vm.tupleStacks[StackValue].Pop()
	amt := vm.data.PopInt64()

	originalAmt := int64(val[2].(Int64))

	if amt >= originalAmt {
		panic(errors.New("split value must be less"))
	}

	vm.tupleStacks[StackValue].Push(Tuple{
		Bytes(ValueTuple),
		historyID(Split, 0, val, Int64(amt)),
		Int64(amt),
		val[3],
	})

	vm.tupleStacks[StackValue].Push(Tuple{
		Bytes(ValueTuple),
		historyID(Split, 1, val, Int64(amt)),
		Int64(originalAmt - amt),
		val[3],
	})
}

func opLock(vm *vm) {
	n := vm.data.PopInt64()
	var values Tuple
	for i := int64(0); i < n; i++ {
		values = append(values, vm.tupleStacks[StackValue].Pop())
	}
	prog := vm.data.PopBytes()

	historyArgs := append(append([]Value{Int64(n)}, values...), Bytes(prog))

	vm.tupleStacks[StackOutput].Push(Tuple{
		Bytes(OutputTuple),
		historyID(Lock, 0, historyArgs...),
		values,
		Bytes(prog),
	})
}

func opRetire(vm *vm) {
	val := vm.tupleStacks[StackValue].Pop()
	vm.tupleStacks[StackRetirement].Push(Tuple{
		Bytes(RetirementTuple),
		historyID(Retire, 0, val),
	})
}

func opAnchor(vm *vm) {
	tuple := vm.data.PopTuple()
	if !checkTuple(tuple, NonceTuple) {
		panic(errors.New("expected nonce tuple"))
	}
	vm.tupleStacks[StackNonce].Push(tuple)
	vm.tupleStacks[StackAnchor].Push(Tuple{
		Bytes(AnchorTuple),
		historyID(Anchor, 0, tuple),
	})
	exec(vm, tuple[1].(Bytes))
}

func opIssue(vm *vm) {
	assetDef := vm.data.PopTuple()
	if !checkTuple(assetDef, AssetDefinitionTuple) {
		panic(errors.New("expected asset definition tuple"))
	}
	amount := vm.data.PopInt64()
	anchor := vm.tupleStacks[StackAnchor].Pop()
	assetID := calcID(assetDef)
	vm.tupleStacks[StackValue].Push(Tuple{
		Bytes(ValueTuple),
		historyID(Issue, 0, assetDef, Int64(amount), anchor),
		Int64(amount),
		Bytes(assetID),
	})
	vm.tupleStacks[StackAnchor].Push(Tuple{
		Bytes(AnchorTuple),
		historyID(Issue, 1, assetDef, Int64(amount), anchor),
	})
	exec(vm, assetDef[2].(Bytes))
}

func opBefore(vm *vm) {
	vm.tupleStacks[StackTimeConstraint].Push(Tuple{Bytes(MaxTimeTuple), Int64(vm.data.PopInt64())})
}

func opAfter(vm *vm) {
	vm.tupleStacks[StackTimeConstraint].Push(Tuple{Bytes(MinTimeTuple), Int64(vm.data.PopInt64())})
}

func opSummarize(vm *vm) {
	if vm.tupleStacks[StackSummary].Len() > 0 {
		panic(errors.New("txheader already created"))
	}
	var historyArgs []Value
	for vm.tupleStacks[StackInput].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackInput].Pop())
	}
	for vm.tupleStacks[StackOutput].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackOutput].Pop())
	}
	for vm.tupleStacks[StackNonce].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackNonce].Pop())
	}
	for vm.tupleStacks[StackRetirement].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackRetirement].Pop())
	}
	for vm.tupleStacks[StackTimeConstraint].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackTimeConstraint].Pop())
	}
	for vm.tupleStacks[StackAnnotation].Len() > 0 {
		historyArgs = append(historyArgs, vm.tupleStacks[StackAnnotation].Pop())
	}

	vm.tupleStacks[StackSummary].Push(Tuple{
		Bytes(SummaryTuple),
		historyID(Summarize, 0, historyArgs...),
	})
}

func historyID(op byte, idx int, vals ...Value) Bytes {
	history := Tuple{
		Bytes([]byte{op}),
		append(Tuple{}, vals...),
		Int64(idx),
	}
	return Bytes(calcID(history))
}

func checkTuple(v Tuple, expected string) bool {
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
