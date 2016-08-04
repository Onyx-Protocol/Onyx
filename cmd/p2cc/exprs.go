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

func (v varref) translate(stk stack, context *context) (*translation, error) {
	varDepth := stk.lookup(string(v))
	if varDepth < 0 {
		return nil, fmt.Errorf("unknown variable %s", string(v))
	}
	var ops string
	if varDepth > 0 {
		ops = fmt.Sprintf("%d PICK", varDepth)
	} else {
		ops = "DUP"
	}
	s := stk.push(typedName{name: string(v)})
	var result translation
	return result.add(ops, s), nil
}

func (v varref) typ(stk stack) int {
	varDepth := stk.lookup(string(v))
	if varDepth < 0 {
		return unknownType
	}
	return stk[varDepth].typ
}

func (v varref) String() string {
	return string(v)
}

func (b binaryExpr) translate(stk stack, context *context) (*translation, error) {
	lhs, err := b.lhs.translate(stk, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating binaryExpr lhs %+v", b.lhs)
	}
	stkWithLHS := stk.push(lhs.finalStackTop())
	rhs, err := b.rhs.translate(stkWithLHS, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating binaryExpr rhs %+v", b.rhs)
	}
	result := lhs.addMany(rhs.steps)
	s := stk.push(typedName{name: fmt.Sprintf("[%s %s %s]", lhs.finalStackTop(), b.op.op, rhs.finalStackTop())})

	var opcodes string
	if b.op.translation == "" {
		// The operator is == or !=.  Use the types of lhs and rhs to
		// determine whether to treat this as numeric (in)equality or
		// bytewise (in)equality.
		var numeric bool
		if b.lhs.typ(stk) == numType && b.rhs.typ(stk) == numType {
			numeric = true
		} else if b.lhs.typ(stk) == numType && b.rhs.typ(stk) == unknownType {
			numeric = true
		} else if b.lhs.typ(stk) == unknownType && b.rhs.typ(stk) == numType {
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

	return result.add(opcodes, s), nil
}

func (b binaryExpr) typ(stk stack) int {
	return b.op.t
}

func (b binaryExpr) String() string {
	return fmt.Sprintf("binaryExpr{lhs: %s, rhs: %s, op: %s}", b.lhs, b.rhs, b.op.op)
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

func (u unaryExpr) translate(stk stack, context *context) (*translation, error) {
	result, err := u.expr.translate(stk, context)
	if err != nil {
		return nil, errors.Wrap(err, "translating unaryExpr %+v", u.expr)
	}
	s := stk.push(typedName{name: fmt.Sprintf("[%s(%s)]", u.op.translation, result.finalStackTop())})
	return result.add(u.op.translation, s), nil
}

func (u unaryExpr) typ(stk stack) int {
	return u.op.typ
}

func (u unaryExpr) String() string {
	return fmt.Sprintf("unaryExpr{expr: %s, op: %s}", u.expr, u.op.op)
}

var calls = []struct {
	name      string
	args, typ int
}{
	{"asset", 0, bytesType},
	{"amount", 0, numType},
	{"program", 0, bytesType},
	{"time", 0, numType},
	{"circulation", 1, numType},
	{"abs", 1, numType},
	{"hash256", 1, bytesType},
	{"checkpredicate", 1, unknownType},
	{"size", 1, numType},
	{"min", 2, numType},
	{"max", 2, numType},
	{"checksig", 3, boolType},
	{"cat", 2, bytesType},
	{"catpushdata", 2, bytesType},
	{"left", 2, bytesType},
	{"right", 2, bytesType},
	{"reserveoutput", 3, boolType},
	{"findoutput", 3, boolType},
	{"substr", 3, bytesType},
}

var errNumArgs = errors.New("number of args")

func (call callExpr) translate(stk stack, context *context) (*translation, error) {
	if call.name == context.currentContract.name {
		return call.contractCall(stk, context, context.currentContract, true)
	}
	for _, c := range context.allContracts {
		if call.name == c.name {
			return call.contractCall(stk, context, c, false)
		}
	}
	for _, c := range calls {
		if call.name == c.name {
			if len(call.actuals) != c.args {
				return nil, errors.Wrap(errNumArgs, "calling %s: expected %d arg(s), got %d", call.name, c.args, len(call.actuals))
			}
			var output *translation
			var argdescs []string
			for i, a := range call.actuals {
				t, err := a.translate(stk, context)
				if err != nil {
					return nil, errors.Wrapf(err, "translating arg %d in call to %s", i, call.name)
				}
				argdesc := t.finalStackTop().name
				output = output.addMany(t.steps)
				stk = stk.push(typedName{name: argdesc})
				argdescs = append(argdescs, argdesc)
			}
			s := stk.dropN(c.args)
			s = s.push(typedName{name: fmt.Sprintf("[%s(%s)]", call.name, strings.Join(argdescs, ", "))})
			opcodes := strings.ToUpper(call.name)
			if call.name == "size" {
				// Special case: SIZE does not consume its argument, so rejigger the stack to get rid of it
				opcodes += " NIP"
			}
			return output.add(opcodes, s), nil
		}
	}
	return nil, fmt.Errorf("unknown function %s", call.name)
}

func (call callExpr) typ(stk stack) int {
	return call.t
}

func (call callExpr) String() string {
	return fmt.Sprintf("callExpr{name: %s, actuals: %s, t: %d}", call.name, call.actuals, call.t)
}

func (call callExpr) contractCall(stk stack, context *context, contract *contract, isSelf bool) (*translation, error) {
	if len(call.actuals) != len(contract.params) {
		return nil, fmt.Errorf("contract %s requires %d param(s), got %d", contract.name, len(contract.params), len(call.actuals))
	}
	stk = stk.push(typedName{name: "[building pkscript]", typ: bytesType})
	b := txscript.AddDataToScript(nil, txscript.ScriptVersion1)
	b = append(b, txscript.OP_DROP)
	var output *translation
	output = output.add(fmt.Sprintf("DATA_%d 0x%s", len(b), hex.EncodeToString(b)), stk)
	var argdescs []string
	for n := len(call.actuals) - 1; n >= 0; n-- {
		actual := call.actuals[n]
		t, err := actual.translate(stk, context)
		if err != nil {
			return nil, errors.Wrapf(err, "translating arg %d in call to %s", n, call.name)
		}
		argdesc := t.finalStackTop().name
		output = output.addMany(t.steps)
		output = output.add("CATPUSHDATA", stk)
		argdescs = append(argdescs, argdesc)
	}
	if len(call.actuals) > 0 {
		b = txscript.AddInt64ToScript(nil, int64(len(call.actuals)))
		b = append(b, txscript.OP_ROLL)
		output = output.add(fmt.Sprintf("DATA_%d 0x%s CAT", len(b), hex.EncodeToString(b)), stk)
	}
	b = []byte{txscript.OP_DUP, txscript.OP_SHA3}
	output = output.add(fmt.Sprintf("DATA_2 0x%s CAT", hex.EncodeToString(b)), stk)

	if isSelf {
		output = output.add("OUTPUTSCRIPT SIZE 34 SUB 32 SUBSTR CATPUSHDATA", stk)
	} else {
		t, err := translate(contract, context.allContracts)
		if err != nil {
			return nil, err
		}
		hash, err := t.getHash()
		if err != nil {
			return nil, err
		}
		b = txscript.AddDataToScript(nil, hash[:])
		output = output.add(fmt.Sprintf("DATA_%d 0x%s CAT", len(b), hex.EncodeToString(b)), stk)
	}

	b = []byte{txscript.OP_EQUALVERIFY, txscript.OP_0, txscript.OP_CHECKPREDICATE}
	// reverse argdescs
	for i := 0; i < len(argdescs)/2; i++ {
		other := len(argdescs) - i - 1
		argdescs[i], argdescs[other] = argdescs[other], argdescs[i]
	}
	s := stk.drop()
	s = s.push(typedName{name: fmt.Sprintf("[%s(%s)]", call.name, strings.Join(argdescs, ", "))})
	output = output.add(fmt.Sprintf("DATA_2 0x%s CAT", hex.EncodeToString(b)), s)
	return output, nil
}

type literal struct {
	b []byte
	t int
}

func (l literal) translate(stk stack, context *context) (*translation, error) {
	ops := string(l.b)
	s := stk.push(typedName{name: fmt.Sprintf("[%s]", string(l.b))})
	var result translation
	return result.add(ops, s), nil
}

func (l literal) typ(stk stack) int {
	return l.t
}

func (l literal) String() string {
	return string(l.b)
}

func newLiteral(b []byte, typ int) *literal {
	return &literal{b: b, t: typ}
}
