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
		result.Params = append(result.Params, ContractParam{Name: param.name, Typ: param.typ})
	}

	for _, clause := range c.clauses {
		info := ClauseInfo{Name: clause.name, Args: []ClauseArg{}}
		// TODO(bobg): this could just be info.Args = clause.params, if we
		// rejigger the types and exports.
		for _, p := range clause.params {
			info.Args = append(info.Args, ClauseArg{Name: p.name, Typ: p.typ})
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

func compileContract(contract *contract, args []ContractArg) ([]byte, error) {
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
		err = compileClause(b, stack, contract, contract.clauses[0])
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
		b2 := newBuilder()
		err = compileClause(b2, stack, contract, clause)
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
			err = compileExpr(b, stack, contract, clause, stmt.expr)
			if err != nil {
				return errors.Wrapf(err, "in verify statement in clause \"%s\"", clause.name)
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
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
				}
				err = compileExpr(b, stack, contract, clause, r)
				if err != nil {
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
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
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
				}
				err = compileExpr(b, stack, contract, clause, r)
				if err != nil {
					return errors.Wrapf(err, "in output statement in clause \"%s\"", clause.name)
				}
				stack = append(stack, stackEntry{})
			}

			// version
			b.addInt64(1)
			stack = append(stack, stackEntry{})

			// prog
			err = compileExpr(b, stack, contract, clause, stmt.call.fn)
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

func compileExpr(b *builder, stack []stackEntry, contract *contract, clause *clause, expr expression) error {
	err := typeCheckExpr(expr)
	if err != nil {
		return err
	}
	switch e := expr.(type) {
	case *binaryExpr:
		err = compileExpr(b, stack, contract, clause, e.left)
		if err != nil {
			return errors.Wrapf(err, "in left operand of \"%s\" expression", e.op.op)
		}
		err = compileExpr(b, append(stack, stackEntry{}), contract, clause, e.right)
		if err != nil {
			return errors.Wrapf(err, "in right operand of \"%s\" expression", e.op.op)
		}
		ops, err := vm.Assemble(e.op.opcodes)
		if err != nil {
			return errors.Wrapf(err, "assembling bytecode in \"%s\" expression", e.op.op)
		}
		b.addRawBytes(ops)

	case *unaryExpr:
		err = compileExpr(b, stack, contract, clause, e.expr)
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
		for i := len(e.args) - 1; i >= 0; i-- {
			a := e.args[i]
			err = compileExpr(b, stack, contract, clause, a)
			if err != nil {
				return errors.Wrapf(err, "compiling argument %d in call expression", i)
			}
			stack = append(stack, stackEntry{})
		}
		ops, err := vm.Assemble(bi.opcodes)
		if err != nil {
			return errors.Wrap(err, "assembling bytecode in call expression")
		}
		b.addRawBytes(ops)

	case *varRef:
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
