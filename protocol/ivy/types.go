package ivy

import "strings"

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
	fn   *ref
	args []expression
}

func (call) iamaExpression() {}

type ref struct {
	names []string

	// Each ref is decorated with the param or the builtin it refers to.
	param   *param
	builtin *builtin
}

func (ref) iamaExpression() {}

func (r ref) String() string {
	return strings.Join(r.names, ".")
}

type integerLiteral int64

func (integerLiteral) iamaExpression() {}

type booleanLiteral bool

func (booleanLiteral) iamaExpression() {}
