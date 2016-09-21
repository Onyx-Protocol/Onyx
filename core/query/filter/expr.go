package filter

import "fmt"

type expr interface {
	String() string
}

type binaryExpr struct {
	op   *binaryOp
	l, r expr
}

func (e binaryExpr) String() string {
	return e.l.String() + " " + e.op.name + " " + e.r.String()
}

type attrExpr struct {
	attr string
}

func (e attrExpr) String() string {
	return e.attr
}

type selectorExpr struct {
	ident   string
	objExpr expr
}

func (e selectorExpr) String() string {
	return e.objExpr.String() + "." + e.ident
}

type parenExpr struct {
	inner expr
}

func (e parenExpr) String() string {
	return "(" + e.inner.String() + ")"
}

type valueExpr struct {
	typ   token
	value string
}

func (e valueExpr) String() string {
	return e.value
}

type envExpr struct {
	ident string
	expr  expr
}

func (e envExpr) String() string {
	return e.ident + "(" + e.expr.String() + ")"
}

type placeholderExpr struct {
	num int
}

func (e placeholderExpr) String() string {
	return fmt.Sprintf("$%d", e.num)
}

// Type defines the value types in filter expressions.
type Type int

const (
	Any Type = iota
	Bool
	String
	Integer
	Object
)

func (t Type) String() string {
	switch t {
	case Any:
		return "any"
	case Bool:
		return "bool"
	case String:
		return "string"
	case Integer:
		return "integer"
	case Object:
		return "object"
	}
	panic("unknown type")
}
