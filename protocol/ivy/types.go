package ivy

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type contract struct {
	name    string
	params  []*param
	value   []lockedValue
	clauses []*clause
}

type param struct {
	name, typ string
}

type clause struct {
	name       string
	params     []*param
	spends     []spentValue
	statements []statement

	// decorations
	mintimes, maxtimes []string
}

type statement interface {
	iamaStatement()
}

type verifyStatement struct {
	expr expression

	// Some verify statements are decorated with pointers to associated
	// output statements. Such verifies don't get compiled themselves,
	// but contribute arguments for use in CHECKOUTPUT.
	associatedOutput *outputStatement
}

func (verifyStatement) iamaStatement() {}

type outputStatement struct {
	call *call

	// The name of the AssetAmount parameter against which the value is
	// checked
	param string

	// Added as a decoration, used by CHECKOUTPUT
	index int64
}

func (outputStatement) iamaStatement() {}

type returnStatement struct {
	expr expression
}

func (returnStatement) iamaStatement() {}

type expression interface {
	String() string
	iamaExpression()
}

type binaryExpr struct {
	left, right expression
	op          *binaryOp
}

func (binaryExpr) iamaExpression() {}

func (e binaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.left, e.op.op, e.right)
}

type unaryExpr struct {
	op   *unaryOp
	expr expression
}

func (unaryExpr) iamaExpression() {}

func (e unaryExpr) String() string {
	return fmt.Sprintf("%s%s", e.op.op, e.expr)
}

type call struct {
	fn   expression
	args []expression
}

func (call) iamaExpression() {}

func (e call) String() string {
	var argStrs []string
	for _, a := range e.args {
		argStrs = append(argStrs, a.String())
	}
	return fmt.Sprintf("%s(%s)", e.fn, strings.Join(argStrs, ", "))
}

type propRef struct {
	expr     expression
	property string
}

func (propRef) iamaExpression() {}

func (p propRef) String() string {
	return fmt.Sprintf("%s.%s", p.expr, p.property)
}

type varRef struct {
	name string

	// decoration
	typ string
}

func (varRef) iamaExpression() {}

func (v varRef) String() string {
	return v.name
}

type bytesLiteral []byte

func (bytesLiteral) iamaExpression() {}

func (e bytesLiteral) String() string {
	return "0x" + hex.EncodeToString([]byte(e))
}

type integerLiteral int64

func (integerLiteral) iamaExpression() {}

func (e integerLiteral) String() string {
	return strconv.FormatInt(int64(e), 10)
}

type booleanLiteral bool

func (booleanLiteral) iamaExpression() {}

func (e booleanLiteral) String() string {
	if e {
		return "true"
	}
	return "false"
}

// TODO(bobg): is it overkill to separate lockedValue and spentValue?

type lockedValue string

func (lockedValue) iamaExpression() {}

func (e lockedValue) String() string {
	return string(e)
}

type spentValue string

func (spentValue) iamaExpression() {}

func (e spentValue) String() string {
	return string(e)
}

func typeOf(expr expression) string {
	switch e := expr.(type) {
	case *binaryExpr:
		return e.op.result

	case *unaryExpr:
		return e.op.result

	case *call:
		b := referencedBuiltin(e.fn)
		if b != nil {
			return b.result
		}
		return ""

	case *propRef:
		t := typeOf(e.expr)
		m := properties[t]
		if m != nil {
			return m[e.property]
		}
		return ""

	case *varRef:
		return e.typ

	case bytesLiteral:
		return "String"

	case integerLiteral:
		return "Integer"

	case booleanLiteral:
		return "Boolean"

	case lockedValue:
		return "Value"

	case spentValue:
		return "Value"
	}

	return ""
}
