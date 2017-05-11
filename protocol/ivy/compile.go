package ivy

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/vm"
)

type (
	CompileResult struct {
		Name    string             `json:"name"`
		Program chainjson.HexBytes `json:"program"`
		Params  []ContractParam    `json:"params"`
		Clauses []ClauseInfo       `json:"clause_info"`
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
	}

	ClauseArg struct {
		Name string `json:"name"`
		Typ  string `json:"type"`
	}

	ValueInfo struct {
		Name        string `json:"name"`
		Program     string `json:"program,omitempty"`
		AssetAmount string `json:"asset_amount,omitempty"`
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
	prog, err := compileContract(c, args)
	if err != nil {
		return CompileResult{}, errors.Wrap(err, "compiling contract")
	}
	result := CompileResult{
		Name:    c.name,
		Program: prog,
		Params:  []ContractParam{},
	}
	for _, param := range c.params {
		result.Params = append(result.Params, ContractParam{Name: param.name, Typ: string(param.typ)})
	}

	for _, clause := range c.clauses {
		info := ClauseInfo{
			Name:     clause.name,
			Args:     []ClauseArg{},
			Mintimes: clause.mintimes,
			Maxtimes: clause.maxtimes,
		}
		if info.Mintimes == nil {
			info.Mintimes = []string{}
		}
		if info.Maxtimes == nil {
			info.Maxtimes = []string{}
		}

		// TODO(bobg): this could just be info.Args = clause.params, if we
		// rejigger the types and exports.
		for _, p := range clause.params {
			info.Args = append(info.Args, ClauseArg{Name: p.name, Typ: string(p.typ)})
		}
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *outputStatement:
				valueInfo := ValueInfo{
					Name: s.call.args[0].String(),
				}
				if s.assetAmount != nil {
					valueInfo.AssetAmount = s.assetAmount.String()
				}
				switch f := s.call.fn.(type) {
				case varRef:
					valueInfo.Program = f.String()
				case *propRef:
					valueInfo.Program = f.String()
				}
				info.Values = append(info.Values, valueInfo)
			case *returnStatement:
				valueInfo := ValueInfo{Name: c.params[len(c.params)-1].name}
				info.Values = append(info.Values, valueInfo)
			}
		}
		result.Clauses = append(result.Clauses, info)
	}
	return result, nil
}

type (
	environ  map[string]envEntry
	envEntry struct {
		t typeDesc
		r role
	}
	role int
)

const (
	roleKeyword role = 1 + iota
	roleBuiltin
	roleContract
	roleContractParam
	roleClause
	roleClauseParam
)

var roleDesc = map[role]string{
	roleKeyword:       "keyword",
	roleBuiltin:       "built-in function",
	roleContract:      "contract",
	roleContractParam: "contract parameter",
	roleClause:        "clause",
	roleClauseParam:   "clause parameter",
}

func compileContract(contract *contract, args []ContractArg) ([]byte, error) {
	if len(contract.clauses) == 0 {
		return nil, fmt.Errorf("empty contract")
	}

	env := make(environ)
	for _, k := range keywords {
		env[k] = envEntry{t: nilType, r: roleKeyword}
	}
	for _, b := range builtins {
		env[b.name] = envEntry{t: nilType, r: roleBuiltin}
	}
	if entry, ok := env[contract.name]; ok {
		return nil, fmt.Errorf("contract name \"%s\" conflicts with %s", contract.name, entry)
	}
	env[contract.name] = envEntry{t: contractType, r: roleContract}
	for _, p := range contract.params {
		if entry, ok := env[p.name]; ok {
			return nil, fmt.Errorf("contract parameter \"%s\" conflicts with %s", p.name, roleDesc[entry.r])
		}
		env[p.name] = envEntry{t: p.typ, r: roleContractParam}
	}
	for _, c := range contract.clauses {
		if entry, ok := env[c.name]; ok {
			return nil, fmt.Errorf("clause \"%s\" conflicts with %s", c.name, roleDesc[entry.r])
		}
		env[c.name] = envEntry{t: nilType, r: roleClause}
	}

	err := requireValueParam(contract)
	if err != nil {
		return nil, err
	}
	err = requireAllParamsUsedInClauses(contract.params, contract.clauses)
	if err != nil {
		return nil, err
	}

	stack := addParamsToStack(nil, contract.params)

	b := newBuilder()
	for _, a := range args {
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

	if len(contract.clauses) == 1 {
		err = compileClause(b, stack, contract, env, contract.clauses[0])
		if err != nil {
			return nil, err
		}
		return b.build()
	}

	endTarget := b.newJumpTarget()
	clauseTargets := make([]int, len(contract.clauses))
	for i := range contract.clauses {
		clauseTargets[i] = b.newJumpTarget()
	}

	if len(stack) > 0 {
		// A clause selector is at the bottom of the stack. Roll it to the
		// top.
		b.addInt64(int64(len(stack)))
		b.addOp(vm.OP_ROLL) // stack: [<clause params> <contract params> <clause selector>]
	}

	// clauses 2..N-1
	for i := len(contract.clauses) - 1; i >= 2; i-- {
		b.addOp(vm.OP_DUP)            // stack: [... <clause selector> <clause selector>]
		b.addInt64(int64(i))          // stack: [... <clause selector> <clause selector> <i>]
		b.addOp(vm.OP_NUMEQUAL)       // stack: [... <clause selector> <i == clause selector>]
		b.addJumpIf(clauseTargets[i]) // stack: [... <clause selector>]
	}

	// clause 1
	b.addJumpIf(clauseTargets[1])

	// no jump needed for clause 0

	for i, clause := range contract.clauses {
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
		err = compileClause(b2, stack, contract, env, clause)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling clause %d", i)
		}
		prog, err := b2.build()
		if err != nil {
			return nil, errors.Wrap(err, "assembling bytecode")
		}
		b.addRawBytes(prog)

		if i < len(contract.clauses)-1 {
			b.addJump(endTarget)
		}
	}
	b.setJumpTarget(endTarget)
	return b.build()
}

func compileClause(b *builder, contractStack []stackEntry, contract *contract, outerEnv environ, clause *clause) error {
	// copy env to leave outerEnv unchanged
	env := make(environ)
	for k, v := range outerEnv {
		env[k] = v
	}
	for _, p := range clause.params {
		if entry, ok := env[p.name]; ok {
			return fmt.Errorf("clause parameter \"%s\" conflicts with %s", p.name, roleDesc[entry.r])
		}
		env[p.name] = envEntry{t: p.typ, r: roleClauseParam}
	}

	err := decorateOutputs(contract, clause, env)
	if err != nil {
		return err
	}
	err = requireAllValuesDisposedOnce(contract, clause)
	if err != nil {
		return err
	}
	err = typeCheckClause(contract, clause, env)
	if err != nil {
		return err
	}
	err = requireAllParamsUsedInClause(clause.params, clause)
	if err != nil {
		return err
	}
	assignIndexes(clause)
	stack := addParamsToStack(nil, clause.params)
	stack = append(stack, contractStack...)
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if stmt.associatedOutput != nil {
				// This verify is associated with an output. It doesn't get
				// compiled; instead it contributes its terms to the output
				// statement's CHECKOUTPUT.
				continue
			}
			err = compileExpr(b, stack, contract, clause, env, stmt.expr)
			if err != nil {
				return errors.Wrapf(err, "in verify statement in clause \"%s\"", clause.name)
			}
			b.addOp(vm.OP_VERIFY)

			// special-casing "verify before(expr)" and "verify after(expr)"
			if c, ok := stmt.expr.(*call); ok && len(c.args) == 1 {
				if b := referencedBuiltin(c.fn); b != nil {
					switch b.name {
					case "before":
						clause.maxtimes = append(clause.maxtimes, c.args[0].String())
					case "after":
						clause.mintimes = append(clause.mintimes, c.args[0].String())
					}
				}
			}

		case *outputStatement:
			// index
			b.addInt64(stmt.index)

			// copy of stack allows stack itself to remain unchanged in the
			// next iteration of the statements loop
			ostack := append(stack, stackEntry(fmt.Sprintf("%d", stmt.index)))

			// refdatahash
			b.addData(nil)
			ostack = append(ostack, stackEntry("''"))

			if stmt.assetAmount == nil {
				// amount
				b.addOp(vm.OP_AMOUNT)
				ostack = append(ostack, stackEntry("<amount>"))

				// asset
				b.addOp(vm.OP_ASSET)
				ostack = append(ostack, stackEntry("<asset>"))
			} else {
				// amount
				r := &propRef{
					expr:     stmt.assetAmount,
					property: "amount",
				}
				err = compileExpr(b, ostack, contract, clause, env, r)
				if err != nil {
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
				}
				ostack = append(ostack, stackEntry(stmt.assetAmount.String()+".amount"))

				// asset
				r.property = "asset"
				err = compileExpr(b, ostack, contract, clause, env, r)
				if err != nil {
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
				}
				ostack = append(ostack, stackEntry(stmt.assetAmount.String()+".asset"))
			}

			// version
			b.addInt64(1)
			ostack = append(ostack, stackEntry("1"))

			// prog
			err = compileExpr(b, ostack, contract, clause, env, stmt.call.fn)
			if err != nil {
				return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
			}

			b.addOp(vm.OP_CHECKOUTPUT)
			b.addOp(vm.OP_VERIFY)

		case *returnStatement:
			if len(clause.statements) == 1 {
				// This is the only statement in the clause, make sure TRUE is
				// on the stack.
				b.addOp(vm.OP_TRUE)
			}
		}
	}
	return nil
}

func compileExpr(b *builder, stack []stackEntry, contract *contract, clause *clause, env environ, expr expression) error {
	err := typeCheckExpr(expr, env)
	if err != nil {
		return err
	}
	switch e := expr.(type) {
	case *binaryExpr:
		err = compileExpr(b, stack, contract, clause, env, e.left)
		if err != nil {
			return errors.Wrapf(err, "in left operand of \"%s\" expression", e.op.op)
		}
		err = compileExpr(b, append(stack, stackEntry(e.left.String())), contract, clause, env, e.right)
		if err != nil {
			return errors.Wrapf(err, "in right operand of \"%s\" expression", e.op.op)
		}
		ops, err := vm.Assemble(e.op.opcodes)
		if err != nil {
			return errors.Wrapf(err, "assembling bytecode in \"%s\" expression", e.op.op)
		}
		b.addRawBytes(ops)

	case *unaryExpr:
		err = compileExpr(b, stack, contract, clause, env, e.expr)
		if err != nil {
			return errors.Wrapf(err, "in \"%s\" expression", e.op.op)
		}
		ops, err := vm.Assemble(e.op.opcodes)
		if err != nil {
			return errors.Wrapf(err, "assembling bytecode in \"%s\" expression", e.op.op)
		}
		b.addRawBytes(ops)

	case *call:
		bi := referencedBuiltin(e.fn)
		if bi == nil {
			return fmt.Errorf("unknown function \"%s\"", e.fn)
		}

		// WARNING WARNING WOOP WOOP
		// special-case hack
		// WARNING WARNING WOOP WOOP
		if bi.name == "checkTxMultiSig" {
			// type checking should have done this for us, but just in case:
			if len(e.args) != 2 {
				// xxx err
			}
			newEntries, err := compileArg(b, stack, contract, clause, env, e.args[1])
			if err != nil {
				return err
			}

			// stack: [... sigM ... sig1 M]

			b.addOp(vm.OP_TOALTSTACK) // stack: [... sigM ... sig1]
			newEntries = newEntries[:len(newEntries)-1]

			b.addOp(vm.OP_TXSIGHASH) // stack: [... sigM ... sig1 txsighash]
			newEntries = append(newEntries, stackEntry("<txsighash>"))

			_, err = compileArg(b, append(stack, newEntries...), contract, clause, env, e.args[0])
			if err != nil {
				return err
			}

			// stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N]

			b.addOp(vm.OP_FROMALTSTACK) // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 N M]
			b.addOp(vm.OP_SWAP)         // stack: [... sigM ... sig1 txsighash pubkeyN ... pubkey1 M N]
			b.addOp(vm.OP_CHECKMULTISIG)
			return nil
		}

		for i := len(e.args) - 1; i >= 0; i-- {
			a := e.args[i]
			newEntries, err := compileArg(b, stack, contract, clause, env, a)
			if err != nil {
				return errors.Wrapf(err, "compiling argument %d in call expression", i)
			}
			stack = append(stack, newEntries...)
		}
		ops, err := vm.Assemble(bi.opcodes)
		if err != nil {
			return errors.Wrap(err, "assembling bytecode in call expression")
		}
		b.addRawBytes(ops)

	case varRef:
		return compileRef(b, stack, e)

	case *propRef:
		return compileRef(b, stack, e)

	case integerLiteral:
		b.addInt64(int64(e))

	case bytesLiteral:
		b.addData([]byte(e))

	case booleanLiteral:
		if e {
			b.addOp(vm.OP_TRUE)
		} else {
			b.addOp(vm.OP_FALSE)
		}

	case listExpr:
		// Lists are excluded here because they disobey the invariant of
		// this function: namely, that it increases the stack size by
		// exactly one. (A list pushes its items and its length on the
		// stack.) But they're OK as function-call arguments because the
		// function (presumably) consumes all the stack items added.
		return fmt.Errorf("encountered list outside of function-call context")
	}
	return nil
}

func compileArg(b *builder, stack []stackEntry, contract *contract, clause *clause, env environ, expr expression) ([]stackEntry, error) {
	var newEntries []stackEntry

	if list, ok := expr.(listExpr); ok {
		for i := 0; i < len(list); i++ {
			elt := list[len(list)-i-1]
			err := compileExpr(b, stack, contract, clause, env, elt)
			if err != nil {
				return nil, err
			}
			newEntry := stackEntry(elt.String())
			newEntries = append(newEntries, newEntry)
			stack = append(stack, newEntry)
		}
		b.addInt64(int64(len(list)))
		newEntries = append(newEntries, stackEntry(fmt.Sprintf("%d", len(list))))
		return newEntries, nil
	}

	err := compileExpr(b, stack, contract, clause, env, expr)
	if err != nil {
		return nil, err
	}
	return []stackEntry{stackEntry(expr.String())}, nil
}

func compileRef(b *builder, stack []stackEntry, ref expression) error {
	for depth := 0; depth < len(stack); depth++ {
		if stack[len(stack)-depth-1].matches(ref) {
			switch depth {
			case 0:
				b.addOp(vm.OP_DUP)
			case 1:
				b.addOp(vm.OP_OVER)
			default:
				b.addInt64(int64(depth))
				b.addOp(vm.OP_PICK)
			}
			return nil
		}
	}
	return fmt.Errorf("undefined reference \"%s\"", ref)
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
