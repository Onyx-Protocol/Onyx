package ivy

import "fmt"

type contract struct {
	name    string
	params  []*param
	clauses []*clause
}

type param struct {
	name, typ string
}

type clause struct {
	name       string
	params     []*param
	statements []statement
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

	// The AssetAmount parameter against which the value is checked
	param *param

	// Added as a decoration, used by CHECKOUTPUT
	index int64
}

func (outputStatement) iamaStatement() {}

type returnStatement struct {
	expr expression
}

func (returnStatement) iamaStatement() {}

type expression interface {
	iamaExpression()
}

type binaryExpr struct {
	left, right expression
	op          string
}

func (binaryExpr) iamaExpression() {}

type partialBinaryExpr struct {
	expr expression
	op   string
}

func (partialBinaryExpr) iamaExpression() {}

type unaryExpr struct {
	op   string
	expr expression
}

func (unaryExpr) iamaExpression() {}

type call struct {
	fn   expression
	args []expression
}

func (call) iamaExpression() {}

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

	// decorations
	param   *param
	builtin *builtin
}

func (varRef) iamaExpression() {}

func (v varRef) String() string {
	return v.name
}

type integerLiteral int64

func (integerLiteral) iamaExpression() {}

type booleanLiteral bool

func (booleanLiteral) iamaExpression() {}

func typeOf(expr expression) string {
	switch e := expr.(type) {
	case *binaryExpr:
		return binaryOps[e.op].result

	case *unaryExpr:
		return unaryOps[e.op].result

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
		if e.param != nil {
			return e.param.typ
		}
		if e.builtin != nil {
			// xxx
		}
		return ""

	case integerLiteral:
		return "Integer"

	case booleanLiteral:
		return "Boolean"
	}
	return ""
}
