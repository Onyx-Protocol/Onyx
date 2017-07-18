package txvm2

import "fmt"

// A "run" is a program and a position in it
type run struct {
	pc   int64
	prog []byte
}

type vm struct {
	bcIDs     [][]byte
	txVersion int64
	runlimit  int64
	extension bool

	run      run
	runstack []run

	stacks [numstacks]stack

	summarized bool
}

type opFuncType func(*vm)

type option func(*vm)

func Validate(tx []byte, txVersion, runlimit int64, o ...option) ([32]byte, bool) {
	defer func() {
		if err := recover(); err != nil {
			if vmerr, ok := err.(vmerror); ok {
				// xxx
			}
		}
	}()

	vm := &vm{
		txVersion: txVersion,
		runlimit:  runlimit,
	}
	for _, o := range o {
		o(vm)
	}
	exec(vm, tx)

	var txid [32]byte

	item, ok := vm.getStack(effectstack).peek(0)
	if !ok {
		return txid, false
	}
	txid, ok = getTxID(item)
	if !ok {
		return txid, false
	}
	if !vm.getStack(entrystack).isEmpty() {
		return txid, false
	}
	// xxx other termination conditions?
	return txid, true
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
	opcode, data, n := decodeInst(vm.run.prog[vm.run.pc:])
	// xxx tracing
	vm.run.pc += n
	switch {
	case isSmallIntOp(opcode):
		vm.push(datastack, vint64(opcode-Op0))
	case int(opcode) >= len(opFuncs):
		panic(fmt.Errorf("invalid opcode %d", opcode))
	default:
		f := opFuncs[opcode]
		if f == nil {
			panic(fmt.Errorf("invalid opcode %d", opcode))
		}
		f(vm)
	}
}

// stack access

func (vm *vm) push(stacknum int, v value) {
	vm.stacks[stacknum].push(v)
}

func (vm *vm) pushBool(stacknum int, b bool) {
	var n vint64
	if b {
		n = 1
	}
	vm.push(stacknum, n)
}

func (vm *vm) pop(stacknum int) value {
	res, ok := vm.stacks[stacknum].pop()
	if !ok {
		panic("stack underflow")
	}
	return res
}

func (vm *vm) popBytes(stacknum int) vbytes {
	v := vm.pop(stacknum)
	s, ok := v.(vbytes)
	if !ok {
		panic(fmt.Errorf("%T is not vbytes", v))
	}
	return s
}

func (vm *vm) popInt64(stacknum int) vint64 {
	v := vm.pop(stacknum)
	n, ok := v.(vint64)
	if !ok {
		panic(fmt.Errorf("%T is not vint64", v))
	}
	return n
}

func (vm *vm) popTuple(stacknum int, name string) tuple {
	v := vm.pop(stacknum)
	if !isNamed(v, name) {
		panic(fmt.Errorf("%T is not a %s", v, name))
	}
	return v.(tuple)
}

func (vm *vm) popBool(stacknum int) bool {
	v := vm.pop(datastack)
	if n, ok := v.(vint64); ok {
		return n != 0
	}
	return true
}

func (vm *vm) peek(stacknum int64) value {
	v, ok := vm.getStack(stacknum).peek(0)
	if !ok {
		panic("stack underflow")
	}
	return v
}

func (vm *vm) peekTuple(stacknum int64, name string) tuple {
	v := vm.peek(stacknum)
	if !isNamed(v, name) {
		panic(fmt.Errorf("%T is not a %s", v, name))
	}
	return v.(tuple)
}

func (vm *vm) getStack(stackID int64) *stack {
	if stackID < 0 || stackID >= int64(len(vm.stacks)) {
		panic(fmt.Errorf("bad stack ID %d", stackID))
	}
	return &vm.stacks[stackID]
}
