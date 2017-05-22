package ivy

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/vm"
)

type (
	CompileResult struct {
		Name    string
		Program chainjson.HexBytes
		Value   string
		Params  []ContractParam
		Clauses []ClauseInfo
		Labels  map[uint32]string
	}

	ContractParam struct {
		Name string `json:"name"`
		Typ  string `json:"type"`
	}

	ClauseInfo struct {
		Name   string      `json:"name"`
		Args   []ClauseArg `json:"args"`
		Values []ValueInfo `json:"value_info"`

		// Mintimes is the stringified form of "x" for any "verify after(x)" in the clause
		Mintimes []string `json:"mintimes"`

		// Maxtimes is the stringified form of "x" for any "verify before(x)" in the clause
		Maxtimes []string `json:"maxtimes"`

		// Records each call to a hash function and the type of the
		// argument passed in
		HashCalls []HashCall `json:"hash_calls"`
	}

	ClauseArg struct {
		Name string `json:"name"`
		Typ  string `json:"type"`
	}

	ValueInfo struct {
		Name    string `json:"name"`
		Program string `json:"program,omitempty"`
		Asset   string `json:"asset,omitempty"`
		Amount  string `json:"amount,omitempty"`
	}
)

type ContractArg struct {
	B *bool               `json:"boolean,omitempty"`
	I *int64              `json:"integer,omitempty"`
	S *chainjson.HexBytes `json:"string,omitempty"`
}

// Compile parses an Ivy contract from the supplied reader and
// produces the compiled bytecode and other analysis.
func Compile(r io.Reader, args []ContractArg) (CompileResult, error) {
	inp, err := ioutil.ReadAll(r)
	if err != nil {
		return CompileResult{}, errors.Wrap(err, "reading input")
	}
	c, err := parse(inp)
	if err != nil {
		return CompileResult{}, errors.Wrap(err, "parse error")
	}

	globalEnv := newEnviron(nil)
	for _, k := range keywords {
		globalEnv.add(k, nilType, roleKeyword)
	}
	for _, b := range builtins {
		globalEnv.add(b.name, nilType, roleBuiltin)
	}
	err = globalEnv.add(c.Name, contractType, roleContract)
	if err != nil {
		return CompileResult{}, err
	}

	prog, labels, err := compileContract(c, args, globalEnv)
	if err != nil {
		return CompileResult{}, errors.Wrap(err, "compiling contract")
	}
	result := CompileResult{
		Name:    c.Name,
		Program: prog,
		Params:  []ContractParam{},
		Value:   c.Value,
		Labels:  labels,
	}
	for _, param := range c.Params {
		result.Params = append(result.Params, ContractParam{Name: param.Name, Typ: string(param.bestType())})
	}

	for _, clause := range c.Clauses {
		info := ClauseInfo{
			Name:      clause.Name,
			Args:      []ClauseArg{},
			Mintimes:  clause.MinTimes,
			Maxtimes:  clause.MaxTimes,
			HashCalls: clause.HashCalls,
		}
		if info.Mintimes == nil {
			info.Mintimes = []string{}
		}
		if info.Maxtimes == nil {
			info.Maxtimes = []string{}
		}

		for _, p := range clause.Params {
			info.Args = append(info.Args, ClauseArg{Name: p.Name, Typ: string(p.bestType())})
		}
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *lockStatement:
				valueInfo := ValueInfo{
					Name:    s.locked.String(),
					Program: s.program.String(),
				}
				if s.locked.String() != c.Value {
					for _, r := range clause.Reqs {
						if s.locked.String() == r.name {
							valueInfo.Asset = r.assetExpr.String()
							valueInfo.Amount = r.amountExpr.String()
							break
						}
					}
				}
				info.Values = append(info.Values, valueInfo)
			case *unlockStatement:
				valueInfo := ValueInfo{Name: c.Value}
				info.Values = append(info.Values, valueInfo)
			}
		}
		result.Clauses = append(result.Clauses, info)
	}
	return result, nil
}

func compileContract(contract *Contract, args []ContractArg, globalEnv *environ) ([]byte, map[uint32]string, error) {
	var err error

	if len(contract.Clauses) == 0 {
		return nil, nil, fmt.Errorf("empty contract")
	}
	env := newEnviron(globalEnv)
	for _, p := range contract.Params {
		err = env.add(p.Name, p.Type, roleContractParam)
		if err != nil {
			return nil, nil, err
		}
	}
	err = env.add(contract.Value, valueType, roleContractValue)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range contract.Clauses {
		err = env.add(c.Name, nilType, roleClause)
		if err != nil {
			return nil, nil, err
		}
	}

	stack := addParamsToStack(nil, contract.Params, true)

	b := newBuilder()
	for i := len(args) - 1; i >= 0; i-- {
		a := args[i]
		switch {
		case a.B != nil:
			var n int64
			if *a.B {
				n = 1
			}
			b.addInt64(n)
		case a.I != nil:
			b.addInt64(*a.I)
		case a.S != nil:
			b.addData(*a.S)
		}
	}

	var prog []byte
	var labels map[uint32]string

	if len(contract.Clauses) == 1 {
		err = compileClause(b, stack, contract, env, contract.Clauses[0])
		if err != nil {
			return nil, nil, err
		}
		prog, err = b.build()
	} else {
		endTarget := b.newJumpTarget()
		clauseTargets := make([]int, len(contract.Clauses))
		for i := range contract.Clauses {
			clauseTargets[i] = b.newJumpTarget()
		}

		if len(stack) > 0 {
			// A clause selector is at the bottom of the stack. Roll it to the
			// top.
			b.addInt64(int64(len(stack)))
			b.addOp(vm.OP_ROLL) // stack: [<clause params> <contract params> <clause selector>]
		}

		// clauses 2..N-1
		for i := len(contract.Clauses) - 1; i >= 2; i-- {
			b.addOp(vm.OP_DUP)            // stack: [... <clause selector> <clause selector>]
			b.addInt64(int64(i))          // stack: [... <clause selector> <clause selector> <i>]
			b.addOp(vm.OP_NUMEQUAL)       // stack: [... <clause selector> <i == clause selector>]
			b.addJumpIf(clauseTargets[i]) // stack: [... <clause selector>]
		}

		// clause 1
		b.addJumpIf(clauseTargets[1]) // consumes the clause selector

		// no jump needed for clause 0

		for i, clause := range contract.Clauses {
			b.setJumpTarget(clauseTargets[i])

			// An inner builder is used for each clause body in order to get
			// any final VERIFY instruction left off, and the bytes of its
			// program appended to the outer builder.
			//
			// (Building the clause in the outer builder, then adding a JUMP
			// to endTarget, would cause the omitted VERIFY to be added.)
			//
			// This only works as long as the inner program contains no jumps,
			// whose absolute addresses would be invalidated by this
			// operation. Luckily we don't generate jumps in clause
			// bodies... yet.
			//
			// TODO(bobg): when we _do_ generate jumps in clause bodies, we'll
			// need a cleverer way to remove the trailing VERIFY.
			b2 := newBuilder()

			if i > 1 {
				// Clauses 0 and 1 have no clause selector on top of the
				// stack. Clauses 2 and later do.
				b2.addOp(vm.OP_DROP)
			}

			err = compileClause(b2, stack, contract, env, clause)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "compiling clause \"%s\"", clause.Name)
			}
			b.addFrom(b2)
			if i < len(contract.Clauses)-1 {
				b.addJump(endTarget)
			}
		}
		b.setJumpTarget(endTarget)
		prog, err = b.build()
		if err != nil {
			return nil, nil, err
		}
		jumpAddrs := b.jumpAddrs()
		labels = make(map[uint32]string)
		labels[jumpAddrs[endTarget]] = "_end"
		for i, targ := range clauseTargets {
			labels[jumpAddrs[targ]] = contract.Clauses[i].Name
		}
	}

	err = prohibitValueParams(contract)
	if err != nil {
		return nil, nil, err
	}
	err = prohibitSigParams(contract)
	if err != nil {
		return nil, nil, err
	}
	err = requireAllParamsUsedInClauses(contract.Params, contract.Clauses)
	if err != nil {
		return nil, nil, err
	}

	return prog, labels, nil
}

func compileClause(b *builder, contractStack []stackEntry, contract *Contract, env *environ, clause *Clause) error {
	var err error

	// copy env to leave outerEnv unchanged
	env = newEnviron(env)
	for _, p := range clause.Params {
		err = env.add(p.Name, p.Type, roleClauseParam)
		if err != nil {
			return err
		}
	}
	for _, req := range clause.Reqs {
		err = env.add(req.name, valueType, roleClauseValue)
		if err != nil {
			return err
		}
	}

	assignIndexes(clause)
	stack := addParamsToStack(nil, clause.Params, false)
	stack = append(stack, contractStack...)

	// a count of the number of times each variable is referenced
	counts := make(map[string]int)
	for _, req := range clause.Reqs {
		req.assetExpr.countVarRefs(counts)
		req.amountExpr.countVarRefs(counts)
	}
	for _, s := range clause.statements {
		s.countVarRefs(counts)
	}

	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			stack, err = compileExpr(b, stack, contract, clause, env, counts, stmt.expr)
			if err != nil {
				return errors.Wrapf(err, "in verify statement in clause \"%s\"", clause.Name)
			}
			b.addOp(vm.OP_VERIFY)
			stack = stack[:len(stack)-1]

			// special-case reporting of certain function calls
			if c, ok := stmt.expr.(*callExpr); ok && len(c.args) == 1 {
				if b := referencedBuiltin(c.fn); b != nil {
					switch b.name {
					case "before":
						clause.MaxTimes = append(clause.MaxTimes, c.args[0].String())
					case "after":
						clause.MinTimes = append(clause.MinTimes, c.args[0].String())
					}
				}
			}

		case *lockStatement:
			// index
			b.addInt64(stmt.index)
			stack = append(stack, stackEntry(strconv.FormatInt(stmt.index, 10)))

			// refdatahash
			b.addData(nil)
			stack = append(stack, stackEntry("''"))

			// TODO: permit more complex expressions for locked,
			// like "lock x+y with foo" (?)

			if stmt.locked.String() == contract.Value {
				// amount
				b.addOp(vm.OP_AMOUNT)
				stack = append(stack, stackEntry("<amount>"))

				// asset
				b.addOp(vm.OP_ASSET)
				stack = append(stack, stackEntry("<asset>"))
			} else {
				var req *ClauseRequirement
				for _, r := range clause.Reqs {
					if stmt.locked.String() == r.name {
						req = r
						break
					}
				}
				if req == nil {
					return fmt.Errorf("unknown value \"%s\" in lock statement in clause \"%s\"", stmt.locked, clause.Name)
				}

				// amount
				stack, err = compileExpr(b, stack, contract, clause, env, counts, req.amountExpr)
				if err != nil {
					return errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}

				// asset
				stack, err = compileExpr(b, stack, contract, clause, env, counts, req.assetExpr)
				if err != nil {
					return errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
				}
			}

			// version
			b.addInt64(1)
			stack = append(stack, stackEntry("1"))

			// prog
			stack, err = compileExpr(b, stack, contract, clause, env, counts, stmt.program)
			if err != nil {
				return errors.Wrapf(err, "in lock statement in clause \"%s\"", clause.Name)
			}

			b.addOp(vm.OP_CHECKOUTPUT)
			b.addOp(vm.OP_VERIFY)

			stack = stack[:len(stack)-6]

		case *unlockStatement:
			if len(clause.statements) == 1 {
				// This is the only statement in the clause, make sure TRUE is
				// on the stack.
				b.addOp(vm.OP_TRUE)
			}
		}
	}

	err = requireAllValuesDisposedOnce(contract, clause)
	if err != nil {
		return err
	}
	err = typeCheckClause(contract, clause, env)
	if err != nil {
		return err
	}
	err = requireAllParamsUsedInClause(clause.Params, clause)
	if err != nil {
		return err
	}

	return nil
}

func compileExpr(b *builder, stack []stackEntry, contract *Contract, clause *Clause, env *environ, counts map[string]int, expr expression) ([]stackEntry, error) {
	var err error

	switch e := expr.(type) {
	case *binaryExpr:
		// Do typechecking after compiling subexpressions (because other
		// compilation errors are more interesting than type mismatch
		// errors).

		stack, err = compileExpr(b, stack, contract, clause, env, counts, e.left)
		if err != nil {
			return nil, errors.Wrapf(err, "in left operand of \"%s\" expression", e.op.op)
		}
		stack, err = compileExpr(b, stack, contract, clause, env, counts, e.right)
		if err != nil {
			return nil, errors.Wrapf(err, "in right operand of \"%s\" expression", e.op.op)
		}

		lType := e.left.typ(env)
		if e.op.left != "" && lType != e.op.left {
			return nil, fmt.Errorf("in \"%s\", left operand has type \"%s\", must be \"%s\"", e, lType, e.op.left)
		}

		rType := e.right.typ(env)
		if e.op.right != "" && rType != e.op.right {
			return nil, fmt.Errorf("in \"%s\", right operand has type \"%s\", must be \"%s\"", e, rType, e.op.right)
		}

		switch e.op.op {
		case "==", "!=":
			if lType != rType {
				// Maybe one is Hash and the other is (more-specific-Hash subtype).
				// TODO(bobg): generalize this mechanism
				if lType == hashType && isHashSubtype(rType) {
					propagateType(contract, clause, env, rType, e.left)
				} else if rType == hashType && isHashSubtype(lType) {
					propagateType(contract, clause, env, lType, e.right)
				} else {
					return nil, fmt.Errorf("type mismatch in \"%s\": left operand has type \"%s\", right operand has type \"%s\"", e, lType, rType)
				}
			}
			if lType == "Boolean" {
				return nil, fmt.Errorf("in \"%s\": using \"%s\" on Boolean values not allowed", e, e.op.op)
			}
		}

		for _, op := range e.op.opcodes {
			b.addOp(op)
		}
		stack = append(stack[:len(stack)-2], stackEntry(e.String()))

	case *unaryExpr:
		// Do typechecking after compiling subexpression (because other
		// compilation errors are more interesting than type mismatch
		// errors).

		var err error
		stack, err = compileExpr(b, stack, contract, clause, env, counts, e.expr)
		if err != nil {
			return nil, errors.Wrapf(err, "in \"%s\" expression", e.op.op)
		}

		if e.op.operand != "" && e.expr.typ(env) != e.op.operand {
			return nil, fmt.Errorf("in \"%s\", operand has type \"%s\", must be \"%s\"", e, e.expr.typ(env), e.op.operand)
		}
		for _, op := range e.op.opcodes {
			b.addOp(op)
		}
		stack = append(stack[:len(stack)-1], stackEntry(e.String()))

	case *callExpr:
		bi := referencedBuiltin(e.fn)
		if bi == nil {
			if e.fn.typ(env) == contractType {
				if e.fn.String() != contract.Name {
					return nil, fmt.Errorf("calling other contracts not yet supported")
				}
				// xxx TODO contract composition
				return nil, nil
			}
			return nil, fmt.Errorf("unknown function \"%s\"", e.fn)
		}

		if len(e.args) != len(bi.args) {
			return nil, fmt.Errorf("wrong number of args for \"%s\": have %d, want %d", bi.name, len(e.args), len(bi.args))
		}

		// WARNING WARNING WOOP WOOP
		// special-case hack
		// WARNING WARNING WOOP WOOP
		if bi.name == "checkTxMultiSig" {
			if _, ok := e.args[0].(listExpr); !ok {
				return nil, fmt.Errorf("checkTxMultiSig expects list literals, got %T for argument 0", e.args[0])
			}
			if _, ok := e.args[1].(listExpr); !ok {
				return nil, fmt.Errorf("checkTxMultiSig expects list literals, got %T for argument 1", e.args[1])
			}

			var err error
			var k1, k2 int

			stack, k1, err = compileArg(b, stack, contract, clause, env, counts, e.args[1])
			if err != nil {
				return nil, err
			}

			// stack: [... sigM ... sig1 M]

			b.addOp(vm.OP_TOALTSTACK) // stack: [... sigM ... sig1]
			altElt := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			b.addOp(vm.OP_TXSIGHASH) // stack: [... sigM ... sig1 txsighash]
			stack = append(stack, stackEntry("<txsighash>"))

			stack, k2, err = compileArg(b, stack, contract, clause, env, counts, e.args[0])
			if err != nil {
				return nil, err
			}

			// stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N]

			b.addOp(vm.OP_FROMALTSTACK) // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N M]
			stack = append(stack, altElt)

			b.addOp(vm.OP_SWAP) // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 M N]
			stack[len(stack)-2], stack[len(stack)-1] = stack[len(stack)-1], stack[len(stack)-2]

			b.addOp(vm.OP_CHECKMULTISIG)
			stack = stack[:len(stack)-k1-k2-1]
			stack = append(stack, stackEntry(e.String()))

			return stack, nil
		}

		var k int

		for i := len(e.args) - 1; i >= 0; i-- {
			a := e.args[i]
			var k2 int
			var err error
			stack, k2, err = compileArg(b, stack, contract, clause, env, counts, a)
			if err != nil {
				return nil, errors.Wrapf(err, "compiling argument %d in call expression", i)
			}
			k += k2
		}

		// Do typechecking after compiling subexpressions (because other
		// compilation errors are more interesting than type mismatch
		// errors).
		for i, actual := range e.args {
			if bi.args[i] != "" && actual.typ(env) != bi.args[i] {
				return nil, fmt.Errorf("argument %d to \"%s\" has type \"%s\", must be \"%s\"", i, bi.name, actual.typ(env), bi.args[i])
			}
		}

		for _, op := range bi.opcodes {
			b.addOp(op)
		}
		stack = stack[:len(stack)-k]
		stack = append(stack, stackEntry(e.String()))

		// special-case reporting
		switch bi.name {
		case "sha3", "sha256":
			clause.HashCalls = append(clause.HashCalls, HashCall{bi.name, e.args[0].String(), string(e.args[0].typ(env))})
		}

	case varRef:
		return compileRef(b, stack, counts, e)

	case integerLiteral:
		b.addInt64(int64(e))
		stack = append(stack, stackEntry(strconv.FormatInt(int64(e), 10)))

	case bytesLiteral:
		b.addData([]byte(e))
		s := hex.EncodeToString([]byte(e))
		if s == "" {
			s = "''"
		}
		stack = append(stack, stackEntry(s))

	case booleanLiteral:
		var s string
		if e {
			b.addOp(vm.OP_TRUE)
			s = "true"
		} else {
			b.addOp(vm.OP_FALSE)
			s = "false"
		}
		stack = append(stack, stackEntry(s))

	case listExpr:
		// Lists are excluded here because they disobey the invariant of
		// this function: namely, that it increases the stack size by
		// exactly one. (A list pushes its items and its length on the
		// stack.) But they're OK as function-call arguments because the
		// function (presumably) consumes all the stack items added.
		return nil, fmt.Errorf("encountered list outside of function-call context")
	}
	return stack, nil
}

func compileArg(b *builder, stack []stackEntry, contract *Contract, clause *Clause, env *environ, counts map[string]int, expr expression) ([]stackEntry, int, error) {
	var n int
	if list, ok := expr.(listExpr); ok {
		for i := 0; i < len(list); i++ {
			elt := list[len(list)-i-1]
			var err error
			stack, err = compileExpr(b, stack, contract, clause, env, counts, elt)
			if err != nil {
				return nil, 0, err
			}
			n++
		}
		b.addInt64(int64(len(list)))
		stack = append(stack, stackEntry(strconv.FormatInt(int64(len(list)), 10)))
		n++
		return stack, n, nil
	}
	var err error
	stack, err = compileExpr(b, stack, contract, clause, env, counts, expr)
	return stack, 1, err
}

func compileRef(b *builder, stack []stackEntry, counts map[string]int, ref varRef) ([]stackEntry, error) {
	for depth := 0; depth < len(stack); depth++ {
		if stack[len(stack)-depth-1].matches(ref) {
			var isFinal bool
			if count, ok := counts[string(ref)]; ok && count > 0 {
				count--
				counts[string(ref)] = count
				isFinal = count == 0
			}

			switch depth {
			case 0:
				if !isFinal {
					b.addOp(vm.OP_DUP)
					stack = append(stack, stackEntry(ref.String()))
				}
			case 1:
				if isFinal {
					b.addOp(vm.OP_SWAP)
					stack[len(stack)-2], stack[len(stack)-1] = stack[len(stack)-1], stack[len(stack)-2]
				} else {
					b.addOp(vm.OP_OVER)
					stack = append(stack, stack[len(stack)-2])
				}
			default:
				b.addInt64(int64(depth))
				if isFinal {
					b.addOp(vm.OP_ROLL)
					entry := stack[len(stack)-depth-1]
					pre := stack[:len(stack)-depth-1]
					post := stack[len(stack)-depth:]
					stack = append(pre, post...)
					stack = append(stack, entry)
				} else {
					b.addOp(vm.OP_PICK)
					stack = append(stack, stack[len(stack)-depth-1])
				}
			}
			return stack, nil
		}
	}
	return nil, fmt.Errorf("undefined reference \"%s\"", ref)
}

func (a *ContractArg) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	if r, ok := m["boolean"]; ok {
		var bval bool
		err = json.Unmarshal(r, &bval)
		if err != nil {
			return err
		}
		a.B = &bval
		return nil
	}
	if r, ok := m["integer"]; ok {
		var ival int64
		err = json.Unmarshal(r, &ival)
		if err != nil {
			return err
		}
		a.I = &ival
		return nil
	}
	r, ok := m["string"]
	if !ok {
		return fmt.Errorf("contract arg must define one of boolean, integer, string")
	}
	var sval chainjson.HexBytes
	err = json.Unmarshal(r, &sval)
	if err != nil {
		return err
	}
	a.S = &sval
	return nil
}
