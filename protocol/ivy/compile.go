package ivy

import (
	"fmt"
	"io"

	chainjson "chain/encoding/json"
	"chain/protocol/vm"
)

type (
	CompileResult struct {
		Program chainjson.HexBytes `json:"program"`
		Clauses []ClauseInfo       `json:"clause_info"`
	}

	ClauseInfo struct {
		Name   string      `json:"name"`
		Args   []ClauseArg `json:"args,omitempty"`
		Values []ValueInfo `json:"value_info"`
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

// Compile parses an Ivy contract from the supplied input source and
// produces the compiled bytecode.
func Compile(r io.Reader) (CompileResult, error) {
	parsed, err := ParseReader("input", r, Debug(false))
	if err != nil {
		return CompileResult{}, err
	}
	c, ok := parsed.(*contract)
	if !ok {
		return CompileResult{}, fmt.Errorf("parse result has type %T, must be *contract", parsed)
	}
	prog, err := compileContract(c)
	if err != nil {
		return CompileResult{}, err
	}
	result := CompileResult{Program: prog}
	for _, clause := range c.clauses {
		info := ClauseInfo{Name: clause.name}
		for _, p := range clause.params {
			switch p.typ {
			case "Value":
				continue
			case "AssetAmount":
				info.Args = append(info.Args, ClauseArg{Name: p.name + ".asset", Typ: p.typ}, ClauseArg{Name: p.name + ".amount", Typ: p.typ})
			default:
				info.Args = append(info.Args, ClauseArg{Name: p.name, Typ: p.typ})
			}
		}
		for _, stmt := range clause.statements {
			switch s := stmt.(type) {
			case *outputStatement:
				valueInfo := ValueInfo{
					Name: s.call.args[0].(*varRef).name,
				}
				if s.param != nil {
					valueInfo.AssetAmount = s.param.name
				}
				switch f := s.call.fn.(type) {
				case *varRef:
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

func compileContract(contract *contract) ([]byte, error) {
	if len(contract.clauses) == 0 {
		return nil, fmt.Errorf("empty contract")
	}

	err := prohibitDuplicateClauseNames(contract)
	if err != nil {
		return nil, err
	}
	err = prohibitDuplicateVars(contract)
	if err != nil {
		return nil, err
	}
	err = requireValueParam(contract)
	if err != nil {
		return nil, err
	}

	stack := addParamsToStack(nil, contract.params)

	if len(contract.clauses) == 1 {
		b := newBuilder()
		err = compileClause(b, stack, contract, contract.clauses[0])
		if err != nil {
			return nil, err
		}
		return b.build()
	}

	b := newBuilder()
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
		b2 := newBuilder()
		err = compileClause(b2, stack, contract, clause)
		if err != nil {
			return nil, err
		}
		prog, err := b2.build()
		if err != nil {
			return nil, err
		}
		b.addRawBytes(prog)
		if i < len(contract.clauses)-1 {
			b.addJump(endTarget)
		}
	}
	b.setJumpTarget(endTarget)
	return b.build()
}

func compileClause(b *builder, contractStack []stackEntry, contract *contract, clause *clause) error {
	err := decorateRefs(contract, clause)
	if err != nil {
		return err
	}
	err = decorateOutputs(contract, clause)
	if err != nil {
		return err
	}
	err = requireAllValuesDisposedOnce(contract, clause)
	if err != nil {
		return err
	}
	err = typeCheckClause(contract, clause)
	if err != nil {
		return err
	}
	assignIndexes(clause)
	stack := addParamsToStack(contractStack, clause.params)
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if stmt.associatedOutput != nil {
				// This verify is associated with an output. It doesn't get
				// compiled; instead it contributes its terms to the output
				// statement's CHECKOUTPUT.
				continue
			}
			err = compileExpr(b, stack, contract, clause, stmt.expr)
			if err != nil {
				return err
			}
			b.addOp(vm.OP_VERIFY)

		case *outputStatement:
			// index
			b.addInt64(stmt.index)
			stack = append(stack, stackEntry{})

			// refdatahash
			b.addData(nil)
			stack = append(stack, stackEntry{})

			p := stmt.param
			if p == nil {
				// amount
				b.addOp(vm.OP_AMOUNT)
				stack = append(stack, stackEntry{})

				// asset
				b.addOp(vm.OP_ASSET)
				stack = append(stack, stackEntry{})
			} else {
				// amount
				// TODO(bobg): this is a bit of a hack; need a cleaner way to
				// introduce new stack references
				r := &propRef{
					expr: &varRef{
						name: stmt.param.name,
					},
					property: "amount",
				}
				err := decorateRefsInExpr(contract, clause, r)
				if err != nil {
					return err
				}
				err = compileExpr(b, stack, contract, clause, r)
				if err != nil {
					return err
				}
				stack = append(stack, stackEntry{})

				// asset
				r = &propRef{
					expr: &varRef{
						name: stmt.param.name,
					},
					property: "asset",
				}
				err = decorateRefsInExpr(contract, clause, r)
				if err != nil {
					return err
				}
				err = compileExpr(b, stack, contract, clause, r)
				if err != nil {
					return err
				}
				stack = append(stack, stackEntry{})
			}

			// version
			b.addInt64(1)
			stack = append(stack, stackEntry{})

			// prog
			err = compileExpr(b, stack, contract, clause, stmt.call.fn)
			if err != nil {
				return err
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

func compileExpr(b *builder, stack []stackEntry, contract *contract, clause *clause, expr expression) error {
	err := typeCheckExpr(expr)
	if err != nil {
		return err
	}
	switch e := expr.(type) {
	case *binaryExpr:
		info, ok := binaryOps[e.op]
		if !ok {
			return fmt.Errorf("unknown operator \"%s\"", e.op)
		}
		err = compileExpr(b, stack, contract, clause, e.left)
		if err != nil {
			return err
		}
		err = compileExpr(b, append(stack, stackEntry{}), contract, clause, e.right)
		if err != nil {
			return err
		}
		ops, err := vm.Assemble(info.opcodes)
		if err != nil {
			return err
		}
		b.addRawBytes(ops)

	case *unaryExpr:
		err = compileExpr(b, stack, contract, clause, e.expr)
		if err != nil {
			return err
		}
		switch e.op {
		case "-":
			b.addOp(vm.OP_NEGATE)
		case "!":
			b.addOp(vm.OP_NOT)
		default:
			return fmt.Errorf("unknown operator \"%s\"", e.op)
		}

	case *call:
		bi := referencedBuiltin(e.fn)
		if bi == nil {
			return fmt.Errorf("unknown function \"%s\"", e.fn)
		}
		for _, a := range e.args {
			err = compileExpr(b, stack, contract, clause, a)
			if err != nil {
				return err
			}
			stack = append(stack, stackEntry{})
		}
		ops, err := vm.Assemble(bi.opcodes)
		if err != nil {
			return err
		}
		b.addRawBytes(ops)

	case *varRef:
		return compileRef(b, stack, e)

	case *propRef:
		return compileRef(b, stack, e)

	case integerLiteral:
		b.addInt64(int64(e))

	case booleanLiteral:
		if e {
			b.addOp(vm.OP_TRUE)
		} else {
			b.addOp(vm.OP_FALSE)
		}
	}
	return nil
}

func compileRef(b *builder, stack []stackEntry, ref expression) error {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].matches(ref) {
			depth := int64(len(stack) - 1 - i)
			switch depth {
			case 0:
				b.addOp(vm.OP_DUP)
			case 1:
				b.addOp(vm.OP_OVER)
			default:
				b.addInt64(depth)
				b.addOp(vm.OP_PICK)
			}
			return nil
		}
	}
	return fmt.Errorf("undefined reference \"%s\"", ref)
}
