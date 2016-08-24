package main

import (
	"fmt"
	"strings"
)

// Values for the expr.typ() function
const (
	unknownType = iota
	numType
	boolType
	bytesType
)

type (
	typedName struct {
		name string
		typ  int
	}

	context struct {
		currentContract *contract
		allContracts    []*contract
	}

	expr interface {
		translatable
		typ(stack) int
		String() string
	}

	decl struct {
		name string
		val  expr
	}

	// Block is the body of an if, while, or clause.  Only in the case
	// of a clause may it have an optional trailing expr.
	block struct {
		decls []*decl
		stmts []translatable
		expr  expr
	}

	clause struct {
		name   string
		params []typedName
		block  *block
	}

	contract struct {
		name        string
		params      []typedName
		clauses     []*clause
		translation *translation // memoized
	}
)

func translate(contract *contract, contracts []*contract) (*translation, error) {
	return contract.translate(nil, &context{currentContract: contract, allContracts: contracts})
}

// The contract translate function expects the stack to be empty.
// Contract declarations are only valid at the top level.
func (c *contract) translate(stk stack, context *context) (*translation, error) {
	if len(stk) > 0 {
		return nil, fmt.Errorf("stack depth is %d but contract must appear at top level", len(stk))
	}

	if c.translation == nil {
		// Actual stack will be:
		//   [BOTTOM] clauseArgN clauseArgN-1 ... clauseArg1 [clauseSelector] contractArgN contractArgN-1 ... contractArg1 [TOP]
		// (Exactly which clauseArgs are present will depend on the clause selected.)
		// Here we assume the clauseSelector will appear on top of the
		// stack; below, we emit some stack manipulations to make it true.
		if len(c.clauses) > 1 {
			// Unnamed clauseSelector at top of stack after the ROLL operation
			// emitted below.
			stk = stk.bottomAdd(typedName{name: "[clause selector]", typ: numType})
		}
		stk = stk.bottomAddMany(c.params)

		// Stack: [BOTTOM] clauseArgN clauseArgN-1 ... clauseArg1 contractArgN contractArgN-1 ... contractArg1 [clauseSelector] [TOP]
		translated0, err := c.clauses[0].translate(stk, context)
		if err != nil {
			return nil, err
		}

		if len(c.clauses) == 1 {
			return translated0, nil
		}

		var result *translation

		if len(c.params) > 0 {
			// gets the clause selector to the top of the stack
			result = result.add(fmt.Sprintf("%d ROLL", len(c.params)), stk)
		}

		result = result.add("DUP 1 NUMEQUAL IF", stk)
		result = result.addMany(translated0.steps)

		for i := 1; i < len(c.clauses); i++ {
			translated, err := c.clauses[i].translate(stk, context)
			if err != nil {
				return nil, err
			}
			result = result.add(fmt.Sprintf("ELSE DUP %d NUMEQUAL IF", i+1), stk)
			result = result.addMany(translated.steps)
		}
		endif := strings.TrimSuffix(strings.Repeat("ENDIF ", len(c.clauses)), " ")
		result = result.add(endif, stk)

		c.translation = result
	}

	return c.translation, nil
}

// initStack produces a depiction of the stack before the first opcode
// runs.
func (c contract) initStackStr() string {
	var strs []string

	for _, p := range c.params {
		strs = append(strs, p.name)
	}
	if len(c.clauses) > 1 {
		strs = append(strs, "[clause selector] ...clause args...")
	} else {
		for _, p := range c.clauses[0].params {
			strs = append(strs, p.name)
		}
	}
	return strings.Join(strs, " ")
}

func (c clause) translate(stk stack, context *context) (*translation, error) {
	return c.block.translate(stk.bottomAddMany(c.params), context)
}

func (b block) translate(stk stack, context *context) (*translation, error) {
	origDepth := len(stk)
	var output *translation
	for _, d := range b.decls {
		t, err := d.val.translate(stk, context)
		if err != nil {
			return nil, err
		}
		output = output.addMany(t.steps)
		stk = stk.push(typedName{name: d.name, typ: d.val.typ(stk)})
	}
	for _, s := range b.stmts {
		t, err := s.translate(stk, context)
		if err != nil {
			return nil, err
		}
		output = output.addMany(t.steps)
	}
	if b.expr != nil {
		e, err := b.expr.translate(stk, context)
		if err != nil {
			return nil, err
		}
		output = output.addMany(e.steps)
	}
	if len(stk) > origDepth {
		var op string
		delta := len(stk) - origDepth
		s := stk.dropN(delta)
		if b.expr == nil {
			// No trailing expr on top of the stack, so drop items from the
			// top
			op = "DROP"
		} else {
			// There is a trailing expr on top of the stack, so drop items
			// while preserving the top
			op = "NIP"
			s = s.push(output.finalStackTop())
		}
		op = strings.TrimSuffix(strings.Repeat(op+" ", delta), " ")
		output = output.add(op, s)
	}

	return output, nil
}
