package txvm2

// A "run" is a program and a position in it
type run struct {
	pc   int
	prog []byte
}

type vm struct {
	runlimit int

	run      run
	runstack []run

	stacks [numstacks]stack
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

	item, ok := vm.stacks[effectstack].top()
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
	if isSmallIntOp(opcode) {
		vm.stacks[datastack].pushInt64(int64(opcode - Op0))
	} else {
		// xxx range check
		f := opFuncs[opcode]
		f(vm)
	}
}

func (vm *vm) push(v value) {
	vm.stacks[datastack].push(v)
}

func (vm *vm) pop() value {
	res, ok := vm.stacks[datastack].pop()
	if !ok {
		panic(xxx)
	}
	return res
}

func (vm *vm) popString() vstring {
	v := vm.pop()
	s, ok := v.(vstring)
	if !ok {
		panic(xxx)
	}
	return s
}
