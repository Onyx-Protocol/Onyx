package txvm2

import (
	"errors"
	"reflect"
	"strings"
)

// A "run" is a program and a position in it
type run struct {
	pc   int64
	prog []byte
}

type OpTracer func(op byte, prog []byte, vm VM)

type VM interface {
	PC() int64
	Stack(int64) Stack
}

type vm struct {
	txVersion                 int64
	initialRunlimit, runlimit int64
	extension                 bool

	run      run
	runstack []run

	stacks [numstacks]stack

	finalized bool

	traceOp    OpTracer
	traceError func(error)
}

func (vm *vm) PC() int64 {
	return vm.run.pc
}

func (vm *vm) Stack(stacknum int64) Stack {
	return vm.getStack(stacknum)
}

type opFuncType func(*vm)

var ErrResidue = errors.New("residue on stack(s)")

func Validate(txprog []byte, txVersion, runlimit int64, o ...Option) (txid [32]byte, err error) {
	vm := &vm{
		txVersion:       txVersion,
		initialRunlimit: runlimit,
		runlimit:        runlimit,
		traceOp:         func(byte, []byte, VM) {},
		traceError:      func(error) {},
	}

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); ok {
				vm.traceError(err)
				return
			}
			// r is some other non-error object, re-panic
			panic(r)
		}
	}()

	for _, o := range o {
		o(vm)
	}

	exec(vm, txprog)

	tx := vm.peekTx(effectstack)
	copy(txid[:], tx.id())
	if !vm.getStack(entrystack).isEmpty() {
		return txid, vm.wraperr(ErrResidue)
	}
	// xxx other termination conditions?
	return txid, nil
}

func exec(vm *vm, prog []byte) {
	if len(vm.run.prog) > 0 {
		vm.runstack = append(vm.runstack, vm.run)
		defer func() {
			vm.run = vm.runstack[len(vm.runstack)-1]
			vm.runstack = vm.runstack[:len(vm.runstack)-1]
		}()
	}
	for vm.run.pc < int64(len(vm.run.prog)) {
		step(vm)
	}
}

func step(vm *vm) {
	opcode := vm.run.prog[vm.run.pc]
	vm.traceOp(opcode, vm.run.prog, vm)
	vm.run.pc++
	switch {
	case isSmallIntOp(opcode):
		vm.push(datastack, vint64(opcode-MinSmallInt))
	case isNop(opcode):
		if !vm.extension {
			panic(vm.errf("invalid opcode %d", opcode))
		}
		return
	default:
		f := opFuncs[opcode]
		f(vm)
	}
}

// stack access

func (vm *vm) push(stacknum int64, v Item) {
	vm.stacks[stacknum].push(v)
}

func (vm *vm) pushBool(stacknum int64, b bool) {
	var n vint64
	if b {
		n = 1
	}
	vm.push(stacknum, n)
}

func (vm *vm) pop(stacknum int64) Item {
	res, ok := vm.stacks[stacknum].pop()
	if !ok {
		panic(vm.err("stack underflow"))
	}
	return res
}

func (vm *vm) popBytes(stacknum int64) []byte {
	v := vm.pop(stacknum)
	s, ok := v.(vbytes)
	if !ok {
		panic(vm.errf("%T is not vbytes", v))
	}
	return []byte(s)
}

func (vm *vm) popInt64(stacknum int64) int64 {
	v := vm.pop(stacknum)
	n, ok := v.(vint64)
	if !ok {
		panic(vm.errf("%T is not vint64", v))
	}
	return int64(n)
}

func (vm *vm) popTuple(stacknum int64, types ...namedtuple) namedtuple {
	v := vm.pop(stacknum)
	t := v.(tuple)
	var names []string
	if len(types) > 0 {
		tupleValue := reflect.ValueOf(t)
		for _, typ := range types {
			names = append(names, typ.name())
			tt := reflect.TypeOf(typ)
			// xxx get base type (tt is *foo, get foo)
			vv := reflect.New(tt)
			detuple := vv.MethodByName("detuple")
			res := detuple.Call([]reflect.Value{tupleValue})
			ok := res[0].Interface().(bool)
			if ok {
				return vv.Interface().(namedtuple)
			}
		}
	}
	panic(vm.errf("tuple is not a %s", strings.Join(names, ", ")))
}

func (vm *vm) popBool(stacknum int64) bool {
	v := vm.pop(datastack)
	if n, ok := v.(vint64); ok {
		return n != 0
	}
	return true
}

func (vm *vm) peek(stacknum int64) Item {
	v, ok := vm.getStack(stacknum).peek(0)
	if !ok {
		panic(vm.err("stack underflow"))
	}
	return v
}

func (vm *vm) peekN(stacknum, n int64) []Item {
	res := vm.getStack(stacknum).peekN(n)
	if int64(len(res)) != n {
		panic(vm.errf("only %d of %d item(s) available", len(res), n))
	}
	return res
}

func (vm *vm) getStack(stackID int64) *stack {
	if stackID < 0 || stackID >= int64(len(vm.stacks)) {
		panic(vm.errf("bad stack ID %d", stackID))
	}
	return &vm.stacks[stackID]
}

func (vm *vm) err(msg string) vmerror {
	return vmerr(vm, msg)
}

func (vm *vm) errf(msg string, arg ...interface{}) vmerror {
	return vmerrf(vm, msg, arg...)
}

func (vm *vm) wraperr(e error) vmerror {
	return vmerror{e: e, vm: vm}
}
