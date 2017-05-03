package ivy

import (
	"fmt"
	"strconv"
)

// Helper routines for keeping inline Go code in grammar.peg to a
// minimum.

func mkContract(name, params, clauses interface{}) (*contract, error) {
	n, ok := name.(string)
	if !ok {
		return nil, fmt.Errorf("mkContract: name has type %T, want string", name)
	}
	p, ok := params.([]*param)
	if !ok {
		return nil, fmt.Errorf("mkContract: params has type %T, want []*param", params)
	}
	c, ok := clauses.([]*clause)
	if !ok {
		return nil, fmt.Errorf("mkContract: clauses has type %Y, want []*clause", clauses)
	}
	return &contract{
		name:    n,
		params:  p,
		clauses: c,
	}, nil
}

func prependParams(first, rest interface{}) ([]*param, error) {
	f, ok := first.([]*param)
	if !ok {
		return nil, fmt.Errorf("prependParams: first has type %T, want []*param", first)
	}
	r, ok := rest.([]*param)
	if !ok {
		return nil, fmt.Errorf("prependParams: rest has type %T, want []*param", rest)
	}
	return append(f, r...), nil
}

func mkParams(first, rest interface{}) ([]*param, error) {
	f, ok := first.(string)
	if !ok {
		return nil, fmt.Errorf("mkParams: first has type %T, want string", first)
	}
	r, ok := rest.([]*param)
	if !ok {
		return nil, fmt.Errorf("mkParams: rest has type %T, want []*param", rest)
	}
	if len(r) == 0 {
		return nil, fmt.Errorf("mkParams: rest is empty")
	}
	return append([]*param{&param{name: f, typ: r[0].typ}}, r...), nil
}

func mkParam(only, typ interface{}) ([]*param, error) {
	o, ok := only.(string)
	if !ok {
		return nil, fmt.Errorf("mkParam: only has type %T, want string", only)
	}
	t, ok := typ.(string)
	if !ok {
		return nil, fmt.Errorf("mkParam: t has type %T, want string", t)
	}
	return []*param{&param{name: o, typ: t}}, nil
}

func prependClause(first, rest interface{}) ([]*clause, error) {
	f, ok := first.(*clause)
	if !ok {
		return nil, fmt.Errorf("prependClause: first has type %T, want *clause", first)
	}
	r, ok := rest.([]*clause)
	if !ok {
		return nil, fmt.Errorf("prependClause: rest has type %T, want []*clause", rest)
	}
	return append([]*clause{f}, r...), nil
}

func mkClauses(only interface{}) ([]*clause, error) {
	o, ok := only.(*clause)
	if !ok {
		return nil, fmt.Errorf("mkClauses: only has type %T, want *clause")
	}
	return []*clause{o}, nil
}

func mkClause(name, params, statements interface{}) (*clause, error) {
	n, ok := name.(string)
	if !ok {
		return nil, fmt.Errorf("mkClause: name has type %T, want string", name)
	}
	p, ok := params.([]*param)
	if !ok {
		return nil, fmt.Errorf("mkClause: params has type %T, want []*param", params)
	}
	s, ok := statements.([]statement)
	if !ok {
		return nil, fmt.Errorf("mkClause: statements has type %T, want []*statement", statements)
	}
	return &clause{
		name:       n,
		params:     p,
		statements: s,
	}, nil
}

func prependStatement(first, rest interface{}) ([]statement, error) {
	f, ok := first.(statement)
	if !ok {
		return nil, fmt.Errorf("prependStatement: first has type %T, want statement", first)
	}
	r, ok := rest.([]statement)
	if !ok {
		return nil, fmt.Errorf("prependStatement: rest has type %T, want []statement", rest)
	}
	return append([]statement{f}, r...), nil
}

func mkStatements(only interface{}) ([]statement, error) {
	o, ok := only.(statement)
	if !ok {
		return nil, fmt.Errorf("mkStatements: only has type %T, want statement", only)
	}
	return []statement{o}, nil
}

func mkVerify(expr interface{}) (*verifyStatement, error) {
	e, ok := expr.(expression)
	if !ok {
		return nil, fmt.Errorf("mkVerify: expr has type %T, want expression", expr)
	}
	return &verifyStatement{expr: e}, nil
}

func mkOutput(callExpr interface{}) (*outputStatement, error) {
	c, ok := callExpr.(*call)
	if !ok {
		return nil, fmt.Errorf("mkOutput: callExpr has type %T, want *call", callExpr)
	}
	return &outputStatement{call: c}, nil
}

func mkReturn(expr interface{}) (*returnStatement, error) {
	e, ok := expr.(expression)
	if !ok {
		return nil, fmt.Errorf("mkReturn: expr has type %T, want expression", expr)
	}
	return &returnStatement{expr: e}, nil
}

func mkPropRef(expr, property interface{}) (*propRef, error) {
	e, ok := expr.(expression)
	if !ok {
		return nil, fmt.Errorf("mkPropRef: expr has type %T, want expression", expr)
	}
	p, ok := property.(string)
	if !ok {
		return nil, fmt.Errorf("mkPropRef: property has type %T, want string", property)
	}
	return &propRef{expr: e, property: p}, nil
}

func mkVarRef(name interface{}) (*varRef, error) {
	n, ok := name.(string)
	if !ok {
		return nil, fmt.Errorf("mkVarRef: name has type %T, want string", name)
	}
	return &varRef{name: n}, nil
}

func mkBinaryExpr(left, op, right interface{}) (*binaryExpr, error) {
	l, ok := left.(expression)
	if !ok {
		return nil, fmt.Errorf("mkBinaryExpr: left has type %T, want expression", left)
	}
	o, ok := op.(string)
	if !ok {
		return nil, fmt.Errorf("mkBinaryExpr: op has type %T, want string", op)
	}
	r, ok := right.(expression)
	if !ok {
		return nil, fmt.Errorf("mkBinaryExpr: right has type %T, want expression", right)
	}
	return &binaryExpr{
		left:  l,
		op:    o,
		right: r,
	}, nil
}

func binaryExprFromPartials(partials, right interface{}) (*binaryExpr, error) {
	p, ok := partials.([]*partialBinaryExpr)
	if !ok {
		return nil, fmt.Errorf("binaryExprFromPartials: partials has type %T, want []*partialBinaryExpr", partials)
	}
	r, ok := right.(expression)
	if !ok {
		return nil, fmt.Errorf("binaryExprFromPartials: right has type %T, want expression", right)
	}
	return binaryExprFromPartialsHelper(p, r), nil
}

func binaryExprFromPartialsHelper(partials []*partialBinaryExpr, right expression) *binaryExpr {
	if len(partials) == 1 {
		return &binaryExpr{
			left:  partials[0].expr,
			op:    partials[0].op,
			right: right,
		}
	}
	last := partials[len(partials)-1]
	left := binaryExprFromPartialsHelper(partials[:len(partials)-1], last.expr)
	return &binaryExpr{
		left:  left,
		op:    last.op,
		right: right,
	}
}

func prependPartialBinaryExpr(first, rest interface{}) ([]*partialBinaryExpr, error) {
	f, ok := first.(*partialBinaryExpr)
	if !ok {
		return nil, fmt.Errorf("prependPartialBinaryExpr: first has type %T, want *partialBinaryExpr", first)
	}
	r, ok := rest.([]*partialBinaryExpr)
	if !ok {
		return nil, fmt.Errorf("prependPartialBinaryExpr: rest has type %T, want []*partialBinaryExpr", rest)
	}
	return append([]*partialBinaryExpr{f}, r...), nil
}

func mkPartialBinaryExprs(only interface{}) ([]*partialBinaryExpr, error) {
	o, ok := only.(*partialBinaryExpr)
	if !ok {
		return nil, fmt.Errorf("mkPartialBinaryExprs: only has type %T, want *partialBinaryExpr", only)
	}
	return []*partialBinaryExpr{o}, nil
}

func mkPartialBinaryExpr(expr, op interface{}) (*partialBinaryExpr, error) {
	e, ok := expr.(expression)
	if !ok {
		return nil, fmt.Errorf("mkPartialBinaryExpr: expr has type %T, want expression", expr)
	}
	o, ok := op.(string)
	if !ok {
		return nil, fmt.Errorf("mkPartialBinaryExpr: op has type %T, want string", op)
	}
	return &partialBinaryExpr{
		expr: e,
		op:   o,
	}, nil
}

func mkUnaryExpr(op, expr interface{}) (*unaryExpr, error) {
	o, ok := op.(string)
	if !ok {
		return nil, fmt.Errorf("mkUnaryExpr: op has type %T, want string", op)
	}
	e, ok := expr.(expression)
	if !ok {
		return nil, fmt.Errorf("mkUnaryExpr: expr has type %T, want expression", expr)
	}
	return &unaryExpr{
		op:   o,
		expr: e,
	}, nil
}

func mkCall(fn, args interface{}) (*call, error) {
	f, ok := fn.(expression)
	if !ok {
		return nil, fmt.Errorf("mkCall: fn has type %T, want expression", fn)
	}
	a, ok := args.([]expression)
	if !ok {
		return nil, fmt.Errorf("mkCall: args has type %T, want []expression", args)
	}
	return &call{
		fn:   f,
		args: a,
	}, nil
}

func prependArg(first, rest interface{}) ([]expression, error) {
	f, ok := first.(expression)
	if !ok {
		return nil, fmt.Errorf("prependArg: first has type %T, want expression", first)
	}
	r, ok := rest.([]expression)
	if !ok {
		return nil, fmt.Errorf("prependArg: rest has type %T, want []expression", rest)
	}
	return append([]expression{f}, r...), nil
}

func mkArgs(only interface{}) ([]expression, error) {
	o, ok := only.(expression)
	if !ok {
		return nil, fmt.Errorf("mkArgs: only has type %T, want expression", only)
	}
	return []expression{o}, nil
}

func mkInteger(text []byte) (integerLiteral, error) {
	num, err := strconv.ParseInt(string(text), 10, 64)
	return integerLiteral(num), err
}

func mkBoolean(text []byte) (booleanLiteral, error) {
	return booleanLiteral(string(text) == "true"), nil
}
