package ivy

import (
	"fmt"
	"io"

	"github.com/davecgh/go-spew/spew"

	"chain/protocol/vm"
)

// Compile parses an Ivy contract from the supplied input source and
// produces the compiled bytecode.
func Compile(r io.Reader) ([]byte, error) {
	c, err := ParseReader("input", r)
	if err != nil {
		return nil, err
	}
	return compile(c.(*contract))
}

func compile(contract *contract) ([]byte, error) {
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

	fmt.Printf("* contract:\n%s", spew.Sdump(contract))

	if len(contract.clauses) == 1 {
		b := newBuilder()
		err = compileClause(b, stack, contract, contract.clauses[0])
		if err != nil {
			return nil, err
		}
		return b.Build()
	}

	b := newBuilder()
	endTarget := b.NewJumpTarget()
	clauseTargets := make([]int, len(contract.clauses))
	for i := range contract.clauses {
		clauseTargets[i] = b.NewJumpTarget()
	}

	if len(stack) > 0 {
		// A clause selector is at the bottom of the stack. Roll it to the
		// top.
		b.AddInt64(int64(len(stack)))
		b.AddOp(vm.OP_ROLL) // stack: [<clause params> <contract params> <clause selector>]
	}

	// clauses 2..N-1
	for i := len(contract.clauses) - 1; i >= 2; i-- {
		b.AddOp(vm.OP_DUP)            // stack: [... <clause selector> <clause selector>]
		b.AddInt64(int64(i))          // stack: [... <clause selector> <clause selector> <i>]
		b.AddOp(vm.OP_NUMEQUAL)       // stack: [... <clause selector> <i == clause selector>]
		b.AddJumpIf(clauseTargets[i]) // stack: [... <clause selector>]
	}

	// clause 1
	b.AddJumpIf(clauseTargets[1])

	// no jump needed for clause 0

	for i, clause := range contract.clauses {
		b.SetJumpTarget(clauseTargets[i])
		b2 := newBuilder()
		err = compileClause(b2, stack, contract, clause)
		if err != nil {
			return nil, err
		}
		prog, err := b2.Build()
		if err != nil {
			return nil, err
		}
		b.AddRawBytes(prog)
		if i < len(contract.clauses)-1 {
			b.AddJump(endTarget)
		}
	}
	b.SetJumpTarget(endTarget)
	return b.Build()
}

func compileClause(b *builder, contractStack []stackEntry, contract *contract, clause *clause) error {
	err := requireAllValuesDisposedOnce(contract, clause)
	if err != nil {
		return err
	}
	err = decorateRefs(contract, clause)
	if err != nil {
		return err
	}
	err = decorateOutputs(contract, clause)
	if err != nil {
		return err
	}
	assignIndexes(clause)
	stack := addParamsToStack(contractStack, clause.params)
	for _, s := range clause.statements {
		switch stmt := s.(type) {
		case *verifyStatement:
			if stmt.associatedOutput != nil {
				// This verify is associated with an output. Instead of
				// compiling it, contribute its terms to the output
				// statement's CHECKOUTPUT.
				continue
			}
			err = compileExpr(b, stack, contract, clause, stmt.expr)
			if err != nil {
				return err
			}
			b.AddOp(vm.OP_VERIFY)

		case *outputStatement:
			// index
			b.AddInt64(stmt.index)
			stack = append(stack, stackEntry{})

			// refdatahash
			b.AddData(nil)
			stack = append(stack, stackEntry{})

			p := stmt.param
			if p == nil {
				// amount
				b.AddOp(vm.OP_AMOUNT)
				stack = append(stack, stackEntry{})

				// asset
				b.AddOp(vm.OP_ASSET)
				stack = append(stack, stackEntry{})
			} else {
				// amount
				err := compileExpr(b, stack, contract, clause, &ref{
					names: []string{stmt.param.name, "amount"},
				})
				if err != nil {
					return err
				}
				stack = append(stack, stackEntry{})

				// asset
				err = compileExpr(b, stack, contract, clause, &ref{
					names: []string{stmt.param.name, "asset"},
				})
				if err != nil {
					return err
				}
				stack = append(stack, stackEntry{})
			}

			// version
			b.AddInt64(1)
			stack = append(stack, stackEntry{})

			// prog
			err = compileExpr(b, stack, contract, clause, stmt.call.fn)
			if err != nil {
				return err
			}

			b.AddOp(vm.OP_CHECKOUTPUT)
			b.AddOp(vm.OP_VERIFY)

		case *returnStatement:
			if !exprReferencesParam(stmt.expr, contract.params[len(contract.params)-1]) {
				fmt.Errorf("expression in return statement must be the contract value parameter")
			}
			// xxx add an OP_TRUE if there are no other statements in the clause?
		}
	}
	return nil
}

func compileExpr(b *builder, stack []stackEntry, contract *contract, clause *clause, expr expression) error {
	switch e := expr.(type) {
	case *binaryExpr:
		err := compileExpr(b, stack, contract, clause, e.left)
		if err != nil {
			return err
		}
		err = compileExpr(b, append(stack, stackEntry{}), contract, clause, e.right)
		if err != nil {
			return err
		}
		switch e.op {
		case "==":
			b.AddOp(vm.OP_EQUAL)
		case "!=":
			b.AddOp(vm.OP_EQUAL)
			b.AddOp(vm.OP_NOT)
		case "<=":
			b.AddOp(vm.OP_LESSTHANOREQUAL)
		case ">=":
			b.AddOp(vm.OP_GREATERTHANOREQUAL)
		case "<":
			b.AddOp(vm.OP_LESSTHAN)
		case ">":
			b.AddOp(vm.OP_GREATERTHAN)
		case "+":
			b.AddOp(vm.OP_ADD)
		case "-":
			b.AddOp(vm.OP_SUB)
		default:
			return fmt.Errorf("unknown operator %s", e.op)
		}
	case *unaryExpr:
		err := compileExpr(b, stack, contract, clause, e.expr)
		if err != nil {
			return err
		}
		switch e.op {
		case "-":
			b.AddOp(vm.OP_NEGATE)
		case "!":
			b.AddOp(vm.OP_NOT)
		default:
			return fmt.Errorf("unknown operator %s", e.op)
		}

	case *call:
		if e.fn.builtin == nil {
			return fmt.Errorf("unknown function %s", e.fn)
		}
		// xxx typechecking
		// xxx check len(args) == arity of function
		for _, a := range e.args {
			err := compileExpr(b, stack, contract, clause, a)
			if err != nil {
				return err
			}
			stack = append(stack, stackEntry{})
		}
		b.AddRawBytes(e.fn.builtin.ops)
	case *ref:
		found := false
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].matches(e) {
				found = true
				depth := int64(len(stack) - 1 - i)
				switch depth {
				case 0:
					b.AddOp(vm.OP_DUP)
				case 1:
					b.AddOp(vm.OP_OVER)
				default:
					b.AddInt64(depth)
					b.AddOp(vm.OP_PICK)
				}
			}
		}
		if !found {
			return fmt.Errorf("undefined reference %s", e.names[0])
		}
	case integerLiteral:
		b.AddInt64(int64(e))
	case booleanLiteral:
		if e {
			b.AddOp(vm.OP_TRUE)
		} else {
			b.AddOp(vm.OP_FALSE)
		}
	}
	return nil
}
