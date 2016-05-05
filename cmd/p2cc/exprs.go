package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"chain/cos/txscript"
	"chain/errors"
)

type (
	binaryOp struct {
		op, translation string
		precedence      int
		canAssign       bool
		t               int
	}
	varref   string
	callExpr struct {
		name    string
		actuals []expr
		t       int
	}
	binaryExpr struct {
		lhs, rhs expr
		op       binaryOp
	}
)

var binaryOps = []*binaryOp{
	// These operators and precedences come from
	// https://golang.org/ref/spec#Operators
	{op: "*", precedence: 5, translation: "MUL", canAssign: true, t: numType},
	{op: "/", precedence: 5, translation: "DIV", canAssign: true, t: numType},
	{op: "%", precedence: 5, translation: "MOD", canAssign: true, t: numType},
	{op: "<<", precedence: 5, translation: "LSHIFT", canAssign: true, t: numType},
	{op: ">>", precedence: 5, translation: "RSHIFT", canAssign: true, t: numType},
	{op: "&", precedence: 5, translation: "AND", canAssign: true},
	{op: "&^", precedence: 5, translation: "INVERT AND", canAssign: true},
	{op: "+", precedence: 4, translation: "ADD", canAssign: true, t: numType},
	{op: "-", precedence: 4, translation: "SUB", canAssign: true, t: numType},
	{op: "|", precedence: 4, translation: "OR", canAssign: true},
	{op: "^", precedence: 4, translation: "XOR", canAssign: true},

	// Translation for these two is handled specially
	{op: "==", precedence: 3, t: boolType},
	{op: "!=", precedence: 3, t: boolType},

	{op: "<", precedence: 3, translation: "LESSTHAN", t: boolType},
	{op: "<=", precedence: 3, translation: "LESSTHANOREQUAL", t: boolType},
	{op: ">", precedence: 3, translation: "GREATERTHAN", t: boolType},
	{op: ">=", precedence: 3, translation: "GREATERTHANOREQUAL", t: boolType},
	{op: "&&", precedence: 2, translation: "BOOLAND", canAssign: true, t: boolType},
	{op: "||", precedence: 1, translation: "BOOLOR", canAssign: true, t: boolType},
}

func (v varref) translate(stack []stackItem, context *context) ([]translation, error) {
	varDepth, err := lookup(string(v), stack)
	if err != nil {
		return nil, errors.Wrap(err, "translating varref %s", string(v))
	}
	var ops string
	if varDepth > 0 {
		ops = fmt.Sprintf("%d PICK", varDepth)
	} else {
		ops = "DUP"
	}
	s := []stackItem{{name: string(v)}}
	s = append(s, stack...)
	return []translation{{ops, s}}, nil
}

func (v varref) typ(stack []stackItem) int {
	varDepth, err := lookup(string(v), stack)
	if err != nil {
		return unknownType
	}
	return stack[varDepth].typ
}

func (b binaryExpr) translate(stack []stackItem, context *context) ([]translation, error) {
	lhs, err := b.lhs.translate(stack, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating binaryExpr lhs %+v", b.lhs)
	}
	stackWithLHS := []stackItem{lhs[len(lhs)-1].stack[0]}
	stackWithLHS = append(stackWithLHS, stack...)
	rhs, err := b.rhs.translate(stackWithLHS, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating binaryExpr rhs %+v", b.rhs)
	}
	result := append(lhs, rhs...)
	s := []stackItem{{name: fmt.Sprintf("[%s %s %s]", lhs[len(lhs)-1].stack[0], b.op.op, rhs[len(rhs)-1].stack[0])}}
	s = append(s, stack...)

	var opcodes string
	if b.op.translation == "" {
		// The operator is == or !=.  Use the types of lhs and rhs to
		// determine whether to treat this as numeric (in)equality or
		// bytewise (in)equality.
		var numeric bool
		if b.lhs.typ(stack) == numType && b.rhs.typ(stack) == numType {
			numeric = true
		} else if b.lhs.typ(stack) == numType && b.rhs.typ(stack) == unknownType {
			numeric = true
		} else if b.lhs.typ(stack) == unknownType && b.rhs.typ(stack) == numType {
			numeric = true
		}
		if numeric {
			if b.op.op == "==" {
				opcodes = "NUMEQUAL"
			} else {
				opcodes = "NUMNOTEQUAL"
			}
		} else {
			if b.op.op == "==" {
				opcodes = "EQUAL"
			} else {
				opcodes = "EQUAL NOT"
			}
		}
	} else {
		opcodes = b.op.translation
	}

	result = append(result, translation{opcodes, s})
	return result, nil
}

func (b binaryExpr) typ(stack []stackItem) int {
	return b.op.t
}

type (
	unaryOp struct {
		op, translation string
		typ             int
	}
	unaryExpr struct {
		expr translatable
		op   unaryOp
	}
)

var unaryOps = []*unaryOp{
	{"-", "NEGATE", numType},
	{"!", "NOT", boolType},
	{"^", "INVERT", bytesType},
}

func (u unaryExpr) translate(stack []stackItem, context *context) ([]translation, error) {
	result, err := u.expr.translate(stack, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating unaryExpr %+v", u.expr)
	}
	s := []stackItem{{name: fmt.Sprintf("[%s(%s)]", u.op.translation, result[len(result)-1].stack[0])}}
	s = append(s, stack...)
	result = append(result, translation{u.op.translation, s})
	return result, nil
}

func (u unaryExpr) typ(stack []stackItem) int {
	return u.op.typ
}

var calls = []struct {
	name      string
	args, typ int
}{
	{"asset", 0, bytesType},
	{"amount", 0, numType},
	{"outputscript", 0, bytesType},
	{"time", 0, numType},
	{"circulation", 1, numType},
	{"abs", 1, numType},
	{"hash256", 1, bytesType},
	{"eval", 1, unknownType},
	{"size", 1, numType},
	{"min", 2, numType},
	{"max", 2, numType},
	{"checksig", 2, boolType},
	{"cat", 2, bytesType},
	{"catpushdata", 2, bytesType},
	{"left", 2, bytesType},
	{"right", 2, bytesType},
	{"reserveoutput", 3, boolType},
	{"findoutput", 3, boolType},
	{"substr", 3, bytesType},
}

var errNumArgs = errors.New("number of args")

func (call callExpr) translate(stack []stackItem, context *context) ([]translation, error) {
	if call.name == context.currentContract.name {
		return call.contractCall(stack, context, context.currentContract, true)
	}
	for _, c := range context.allContracts {
		if call.name == c.name {
			return call.contractCall(stack, context, c, false)
		}
	}
	for _, c := range calls {
		if call.name == c.name {
			if len(call.actuals) != c.args {
				return nil, errors.Wrap(errNumArgs, "calling %s: expected %d arg(s), got %d", call.name, c.args, len(call.actuals))
			}
			var output []translation
			var argdescs []string
			for i, a := range call.actuals {
				t, err := a.translate(stack, context)
				if err != nil {
					return nil, errors.Wrap(err, "translating arg %d in call to %s", i, call.name)
				}
				argdesc := t[len(t)-1].stack[0].name
				output = append(output, t...)
				stack = append([]stackItem{{name: argdesc}}, stack...)
				argdescs = append(argdescs, argdesc)
			}
			s := []stackItem{{name: fmt.Sprintf("[%s(%s)]", call.name, strings.Join(argdescs, ", "))}}
			s = append(s, stack[c.args:]...)
			opcodes := strings.ToUpper(call.name)
			if call.name == "size" {
				// Special case: SIZE does not consume its argument, so rejigger the stack to get rid of it
				opcodes += " NIP"
			}
			output = append(output, translation{opcodes, s})
			return output, nil
		}
	}
	return nil, fmt.Errorf("unknown function %s", call.name)
}

func (call callExpr) typ(stack []stackItem) int {
	return call.t
}

func (call callExpr) contractCall(stack []stackItem, context *context, contract *contract, isSelf bool) ([]translation, error) {
	if len(call.actuals) != len(contract.params) {
		return nil, fmt.Errorf("contract %s requires %d param(s), got %d", contract.name, len(contract.params), len(call.actuals))
	}
	stack = append([]stackItem{{name: "[building pkscript]", typ: bytesType}}, stack...)
	b := txscript.AddDataToScript(nil, txscript.ScriptVersion1)
	b = append(b, txscript.OP_DROP)
	output := []translation{{ops: fmt.Sprintf("DATA_%d 0x%s", len(b), hex.EncodeToString(b)), stack: stack}}
	var argdescs []string
	for n := len(call.actuals) - 1; n >= 0; n-- {
		actual := call.actuals[n]
		t, err := actual.translate(stack, context)
		if err != nil {
			return nil, errors.Wrap(err, "translating arg %d in call to %s", n, call.name)
		}
		argdesc := t[len(t)-1].stack[0].name
		output = append(output, t...)
		output = append(output, translation{ops: "CATPUSHDATA", stack: stack})
		argdescs = append(argdescs, argdesc)
	}
	if len(call.actuals) > 0 {
		b = txscript.AddInt64ToScript(nil, int64(len(call.actuals)))
		b = append(b, txscript.OP_ROLL)
		output = append(output, translation{ops: fmt.Sprintf("DATA_%d 0x%s CAT", len(b), hex.EncodeToString(b)), stack: stack})
	}
	b = []byte{txscript.OP_DUP, txscript.OP_HASH256}
	output = append(output, translation{ops: fmt.Sprintf("DATA_2 0x%s CAT", hex.EncodeToString(b)), stack: stack})

	if isSelf {
		output = append(output, translation{ops: "OUTPUTSCRIPT SIZE 34 SUB 32 SUBSTR CATPUSHDATA", stack: stack})
	} else {
		t, err := translate(contract, context.allContracts)
		if err != nil {
			return nil, err
		}
		hash, err := translationToContractHash(t)
		if err != nil {
			return nil, err
		}
		b = txscript.AddDataToScript(nil, hash[:])
		output = append(output, translation{ops: fmt.Sprintf("DATA_%d 0x%s CAT", len(b), hex.EncodeToString(b)), stack: stack})
	}

	b = []byte{txscript.OP_EQUALVERIFY, txscript.OP_EVAL}
	// reverse argdescs
	for i := 0; i < len(argdescs)/2; i++ {
		other := len(argdescs) - i - 1
		argdescs[i], argdescs[other] = argdescs[other], argdescs[i]
	}
	s := []stackItem{{name: fmt.Sprintf("[%s(%s)]", call.name, strings.Join(argdescs, ", "))}}
	s = append(s, stack[1:]...)
	output = append(output, translation{ops: fmt.Sprintf("DATA_2 0x%s CAT", hex.EncodeToString(b)), stack: s})
	return output, nil
}

type literal struct {
	b []byte
	t int
}

func (l literal) translate(stack []stackItem, context *context) ([]translation, error) {
	ops := string(l.b)
	s := []stackItem{{name: fmt.Sprintf("[%s]", string(l.b))}}
	s = append(s, stack...)
	return []translation{{ops, s}}, nil
}

func (l literal) typ(stack []stackItem) int {
	return l.t
}

func newLiteral(b []byte, typ int) *literal {
	return &literal{b: b, t: typ}
}
