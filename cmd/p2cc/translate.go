package main

import (
	"errors"
	"fmt"
	"strings"

	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/hash256"
)

// Values for the expr.typ() function
const (
	unknownType = iota
	numType
	boolType
	bytesType
)

type (
	stackItem struct {
		name string
		typ  int
	}

	translation struct {
		ops   string
		stack []stackItem
	}

	context struct {
		currentContract *contract
		allContracts    []*contract
	}

	translatable interface {
		// The translate() function takes a representation of the stack as
		// an argument, for resolving variable references.  stack[0] is
		// the top of the stack.  It also takes a "context" containing all
		// known parsed contracts, for resolving contract calls.
		//
		// Statements, decls, and exprs are all translatable.  Statements
		// produce code that leaves the stack unchanged.  Decls add one
		// named item to the top of the stack (for the scope of the clause
		// in which they appear).  Exprs add one unnamed item to the top
		// of the stack.
		translate([]stackItem, *context) ([]translation, error)
	}

	expr interface {
		translatable
		typ([]stackItem) int
	}

	decl struct {
		name string
		val  expr
		typ  int
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
		params []stackItem
		block  *block
	}

	contract struct {
		name        string
		params      []stackItem
		clauses     []*clause
		translation []translation // memoized
	}
)

func translate(contract *contract, contracts []*contract) ([]translation, error) {
	return contract.translate(nil, &context{currentContract: contract, allContracts: contracts})
}

func (contract *contract) translate(stack []stackItem, context *context) ([]translation, error) {
	if len(stack) > 0 {
		return nil, fmt.Errorf("stack depth is %d but contract must appear at top level", len(stack))
	}

	if contract.translation == nil {
		// Actual stack will be:
		//   [BOTTOM] clauseArgN clauseArgN-1 ... clauseArg1 [clauseSelector] contractArgN contractArgN-1 ... contractArg1 [TOP]
		// (Exactly which clauseArgs are present will depend on the clause selected.)
		// Here we assume the clauseSelector will appear on top of the
		// stack; below, we emit some stack manipulations to make it true.
		if len(contract.clauses) > 1 {
			// Unnamed clauseSelector at top of stack after the ROLL operation
			// emitted below.
			stack = append(stack, stackItem{name: "[clause selector]", typ: numType})
		}
		stack = append(stack, contract.params...)

		// Stack: [BOTTOM] clauseArgN clauseArgN-1 ... clauseArg1 contractArgN contractArgN-1 ... contractArg1 [clauseSelector] [TOP]
		translated0, err := contract.clauses[0].translate(stack, context)
		if err != nil {
			return nil, err
		}

		if len(contract.clauses) == 1 {
			return translated0, nil
		}

		var result []translation

		if len(contract.params) > 0 {
			// gets the clause selector to the top of the stack
			result = append(result, translation{fmt.Sprintf("%d ROLL", len(contract.params)), stack})
		}

		result = append(result, translation{"DUP 1 NUMEQUAL IF", stack})
		result = append(result, translated0...)

		for i := 1; i < len(contract.clauses); i++ {
			translated, err := contract.clauses[i].translate(stack, context)
			if err != nil {
				return nil, err
			}
			result = append(result, translation{fmt.Sprintf("ELSE DUP %d NUMEQUAL IF", i+1), stack})
			result = append(result, translated...)
		}
		endif := strings.TrimSuffix(strings.Repeat("ENDIF ", len(contract.clauses)), " ")
		result = append(result, translation{endif, stack})

		contract.translation = result
	}

	return contract.translation, nil
}

func (c clause) translate(stack []stackItem, context *context) ([]translation, error) {
	t, err := c.block.translate(append(stack, c.params...), context)
	if err != nil {
		return nil, err
	}
	if c.block.expr == nil {
		t = append(t, translation{"TRUE", append([]stackItem{{name: "TRUE"}}, stack...)})
	}
	return t, nil
}

func (b block) translate(stack []stackItem, context *context) ([]translation, error) {
	origDepth := len(stack)
	var output []translation
	for _, d := range b.decls {
		t, err := d.val.translate(stack, context)
		if err != nil {
			return nil, err
		}
		output = append(output, t...)
		stack = append([]stackItem{{name: d.name, typ: d.val.typ(stack)}}, stack...)
	}
	for _, s := range b.stmts {
		t, err := s.translate(stack, context)
		if err != nil {
			return nil, err
		}
		output = append(output, t...)
	}
	if b.expr != nil {
		e, err := b.expr.translate(stack, context)
		if err != nil {
			return nil, err
		}
		output = append(output, e...)
	}
	if len(stack) > origDepth {
		var (
			op string
			s  []stackItem
		)
		delta := len(stack) - origDepth
		if b.expr == nil {
			// No trailing expr on top of the stack, so drop items from the
			// top
			op = "DROP"
		} else {
			// There is a trailing expr on top of the stack, so drop items
			// while preserving the top
			op = "NIP"
			s = append(s, output[len(output)-1].stack[0])
		}
		s = append(s, stack[delta:]...)
		op = strings.TrimSuffix(strings.Repeat(op+" ", delta), " ")
		output = append(output, translation{ops: op, stack: s})
	}

	return output, nil
}

var errNotFound = errors.New("not found")

func lookup(name string, stack []stackItem) (int, error) {
	for i, entry := range stack {
		if name == entry.name {
			return i, nil
		}
	}
	return 0, errNotFound
}

func (s stackItem) String() string {
	return s.name
}

func translationToContractHash(translation []translation) (bc.ContractHash, error) {
	var allOps []string
	for _, t := range translation {
		allOps = append(allOps, t.ops)
	}
	parsed, err := txscript.ParseScriptString(strings.Join(allOps, " "))
	if err != nil {
		return bc.ContractHash{}, err
	}
	return hash256.Sum(parsed), nil
}
