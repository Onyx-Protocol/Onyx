package chql

import (
	"fmt"
	"strconv"
)

// Query represents a parsed CjQL expression.
type Query struct {
	expr       expr
	Parameters int
}

// String returns a cleaned, canonical representation of the
// ChQL query string.
func (q Query) String() string {
	return q.expr.String()
}

// MarshalText implements the encoding.TextMarshaler interface and
// returns a cleaned, canonical representation of the ChQL query.
func (q Query) MarshalText() ([]byte, error) {
	return []byte(q.expr.String()), nil
}

// Parse parses a query and returns an internal representation of the
// query or an error if it fails to parse.
func Parse(query string) (q Query, err error) {
	expr, parser, err := parse(query)
	if err != nil {
		return q, err
	}
	err = typeCheck(expr, nil)
	if err != nil {
		return q, err
	}

	q = Query{
		Parameters: parser.maxPlaceholder,
		expr:       expr,
	}
	return q, err
}

func parse(exprString string) (expr expr, parser *parser, err error) {
	defer func() {
		r := recover()
		if perr, ok := r.(parseError); ok {
			err = perr
		} else if r != nil {
			panic(r)
		}
	}()

	parser = newParser([]byte(exprString))
	expr = parseExpr(parser)
	parser.parseTok(tokEOF)
	return expr, parser, err
}

func newParser(src []byte) *parser {
	p := new(parser)
	p.scanner.init(src)
	p.next() // advance onto the first input token
	return p
}

// The parser structure holds the parser's internal state.
type parser struct {
	scanner scanner

	maxPlaceholder int

	// Current token
	pos int    // token position
	tok token  // one token look-ahead
	lit string // token literal
}

func determineBinaryOp(p *parser, minPrecedence int) (op *binaryOp, ok bool) {
	op, ok = binaryOps[p.lit]
	return op, ok && op.precedence >= minPrecedence
}

// next advances to the next token.
func (p *parser) next() {
	p.pos, p.tok, p.lit = p.scanner.Scan()
}

func (p *parser) parseLit(lit string) {
	if p.lit != lit {
		p.errorf("got %s, expected %s", p.lit, lit)
	}
	p.next()
}

func (p *parser) parseTok(tok token) {
	if p.tok != tok {
		p.errorf("got %s, expected %s", p.lit, tok.String())
	}
	p.next()
}

func parseExpr(p *parser) expr {
	// Uses the precedence-climbing algorithm:
	// https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method
	expr := parseUnaryExpr(p)
	return parseExprCont(p, expr, 0)
}

func parseUnaryExpr(p *parser) expr {
	// Only one unary expr, NOT <expr>
	if p.lit != "NOT" {
		return parsePrimaryExpr(p)
	}
	p.next()
	expr := parseUnaryExpr(p)
	return notExpr{inner: expr}
}

func parseExprCont(p *parser, lhs expr, minPrecedence int) expr {
	for {
		op, ok := determineBinaryOp(p, minPrecedence)
		if !ok {
			break
		}
		p.next()

		rhs := parseUnaryExpr(p)

		for {
			op2, ok := determineBinaryOp(p, op.precedence+1)
			if !ok {
				break
			}
			rhs = parseExprCont(p, rhs, op2.precedence)
		}
		lhs = binaryExpr{l: lhs, r: rhs, op: op}
	}
	return lhs
}

func parsePrimaryExpr(p *parser) expr {
	x := parseOperand(p)
	for p.lit == "." {
		x = parseSelectorExpr(p, x)
	}
	return x
}

func parseOperand(p *parser) expr {
	switch {
	case p.lit == "(":
		p.next()
		expr := parseExpr(p)
		p.parseLit(")")
		return parenExpr{inner: expr}
	case p.tok == tokString:
		v := valueExpr{typ: p.tok, value: p.lit}
		p.next()
		return v
	case p.tok == tokInteger:
		// Parse the literal into an integer so that we store the string
		// representation of the *decimal* value, never the hex.
		integer, err := strconv.ParseInt(p.lit, 0, 64)
		if err != nil {
			// can't happen; scanner guarantees it
			p.errorf("invalid integer: %q", p.lit)
		}
		v := valueExpr{typ: p.tok, value: strconv.Itoa(int(integer))}
		p.next()
		return v
	case p.tok == tokPlaceholder:
		num, err := strconv.Atoi(p.lit[1:])
		if err != nil || num <= 0 {
			p.errorf("invalid placeholder: %q", p.lit)
		}
		v := placeholderExpr{num: num}
		p.next()

		if num > p.maxPlaceholder {
			p.maxPlaceholder = num
		}
		return v
	default:
		return parseEnvironmentExpr(p)
	}
}

func parseSelectorExpr(p *parser, objExpr expr) expr {
	p.next() // move past the '.'

	ident := p.lit
	p.parseTok(tokIdent)
	return selectorExpr{
		ident:   ident,
		objExpr: objExpr,
	}
}

func parseEnvironmentExpr(p *parser) expr {
	name := p.lit
	p.parseTok(tokIdent)
	if p.lit != "(" {
		return attrExpr{attr: name}
	}
	p.next()
	expr := parseExpr(p)
	p.parseLit(")")
	return envExpr{
		ident: name,
		expr:  expr,
	}
}

type parseError struct {
	pos int
	msg string
}

func (err parseError) Error() string {
	return fmt.Sprintf("col %d: %s", err.pos, err.msg)
}

func (p *parser) errorf(format string, args ...interface{}) {
	panic(parseError{pos: p.pos, msg: fmt.Sprintf(format, args...)})
}
