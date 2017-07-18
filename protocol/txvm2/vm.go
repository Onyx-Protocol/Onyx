package txvm2

// A "run" is a program and a position in it
type run struct {
	pc   int64
	prog []byte
}

type vm struct {
	bcIDs    [][]byte
	runlimit int

	run      run
	runstack []run

	stacks [numstacks]stack

	summarized bool
}

type opFuncType func(*vm)

func Validate(tx []byte, o ...option) ([32]byte, bool) {
	defer func() {
		if err := recover(); err != nil {
			if vmerr, ok := err.(vmerror); ok {
				// xxx
			}
		}
	}()

	vm := &vm{
		runlimit: initialRunLimit,
	}
	for _, o := range o {
		o(vm)
	}
	exec(vm, tx)

	var txid [32]byte

	item, ok := vm.stacks[effectstack].peek()
	if !ok {
		return txid, false
	}
	txid, ok = getTxID(item)
	if !ok {
		return txid, false
	}
	if !vm.stacks[entrystack].isEmpty() {
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
	for vm.pc < len(prog) {
		step(vm)
	}
}

func step(vm *vm) {
	opcode, data, n := decodeInst(vm.prog[vm.pc:])
	// xxx tracing
	vm.pc += n
	switch {
	case isSmallIntOp(opcode):
		vm.pushInt64(datastack, int64(opcode-Op0))
	case opcode >= len(opFuncs):
		panic(xxx)
	default:
		f := opFuncs[opcode]
		if f == nil {
			panic(xxx)
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
		panic(xxx)
	}
	return res
}

func (vm *vm) popBytes(stacknum int) vbytes {
	v := vm.pop(stacknum)
	s, ok := v.(vbytes)
	if !ok {
		panic(xxx)
	}
	return s
}

func (vm *vm) popInt64(stacknum int) vint64 {
	v := vm.pop(stacknum)
	n, ok := v.(vint64)
	if !ok {
		panic(xxx)
	}
	return n
}

func (vm *vm) popTuple(stacknum int, name string) tuple {
	v := vm.pop(stacknum)
	if !isNamed(v, name) {
		panic(xxx)
	}
	return v.(tuple)
}

func (vm *vm) popBool(stacknum int) bool {
	v := vm.pop()
	if n, ok := v.(vint64); ok {
		return n != 0
	}
	return true
}

func (vm *vm) peek(stacknum int) value {
	v, ok := vm.stacks[stacknum].peek()
	if !ok {
		panic(xxx)
	}
	return v
}

func (vm *vm) peekTuple(stacknum int, name string) tuple {
	v := vm.peek(stacknum)
	if !isNamed(v, name) {
		panic(xxx)
	}
	return v.(tuple)
}
