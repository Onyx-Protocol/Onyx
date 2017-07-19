package txvm2

import "fmt"

// A "run" is a program and a position in it
type run struct {
	pc   int64
	prog []byte
}

type vm struct {
	txVersion                 int64
	initialRunlimit, runlimit int64
	extension                 bool

	run      run
	runstack []run

	stacks [numstacks]stack

	finalized bool
}

type opFuncType func(*vm)

type option func(*vm)

func Validate(txprog []byte, txVersion, runlimit int64, o ...option) ([32]byte, bool) {
	defer func() {
		if err := recover(); err != nil {
			// xxx
		}
	}()

	vm := &vm{
		txVersion:       txVersion,
		initialRunlimit: runlimit,
		runlimit:        runlimit,
	}
	for _, o := range o {
		o(vm)
	}
	exec(vm, txprog)

	var txid [32]byte

	item, ok := vm.getStack(effectstack).peek(0)
	if !ok {
		return txid, false
	}
	if !isNamed(item, transactionTuple) {
		return txid, false
	}
	copy(txid[:], getID(item))
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
	opcode := vm.run.prog[vm.run.pc]
	// xxx tracing
	vm.run.pc++
	switch {
	case isSmallIntOp(opcode):
		vm.push(datastack, vint64(opcode-Op0))
	case int(opcode) >= len(opFuncs):
		// NOP instruction
		if !vm.extension {
			panic(fmt.Errorf("invalid opcode %d", opcode))
		}
		return
	default:
		f := opFuncs[opcode]
		if f == nil {
			// NOP instruction
			if !vm.extension {
				panic(fmt.Errorf("invalid opcode %d", opcode))
			}
			return
		}
		f(vm)
	}
}

// stack access

func (vm *vm) push(stacknum int, v item) {
	vm.stacks[stacknum].push(v)
}

func (vm *vm) pushBool(stacknum int, b bool) {
	var n vint64
	if b {
		n = 1
	}
	vm.push(stacknum, n)
}

func (vm *vm) pop(stacknum int) item {
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

func (vm *vm) popTuple(stacknum int, names ...string) tuple {
	v := vm.pop(stacknum)
	if len(names) > 0 {
		ok := false
		for _, name := range names {
			if isNamed(v, name) {
				ok = true
				break
			}
		}
		if !ok {
			panic(fmt.Errorf("%T is not one of %v", v, names))
		}
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

func (vm *vm) peek(stacknum int64) item {
	v, ok := vm.getStack(stacknum).peek(0)
	if !ok {
		panic("stack underflow")
	}
	return v
}

func (vm *vm) peekN(stacknum, n int64) []item {
	res := vm.getStack(stacknum).peekN(n)
	if int64(len(res)) != n {
		panic(fmt.Errorf("only %d of %d item(s) available", len(res), n))
	}
	return res
}

func (vm *vm) peekTuple(stacknum int64, name string) tuple {
	v := vm.peek(stacknum)
	if !isNamed(v, name) {
		panic(fmt.Errorf("%T is not a %s", v, name))
	}
	return v.(tuple)
}

func (vm *vm) peekNTuple(stacknum, n int64, name string) []tuple {
	var res []tuple
	vals := vm.peekN(stacknum, n)
	for i, v := range vals {
		if !isNamed(v, name) {
			panic(fmt.Errorf("item %d, a %T, is not a %s", i, v, name))
		}
		res = append(res, v.(tuple))
	}
	return res
}

func (vm *vm) getStack(stackID int64) *stack {
	if stackID < 0 || stackID >= int64(len(vm.stacks)) {
		panic(fmt.Errorf("bad stack ID %d", stackID))
	}
	return &vm.stacks[stackID]
}
