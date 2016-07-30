package cql

import "fmt"

type expr interface {
	String() string
}

type notExpr struct {
	inner expr
}

func (e notExpr) String() string {
	return "NOT " + e.inner.String()
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
