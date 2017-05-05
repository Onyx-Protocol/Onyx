package ivy

import (
	"bytes"
	"fmt"
	"unicode"
)

// We have some function naming conventions.
//
// For terminals:
//   scanX     takes buf and position, returns new position (and maybe a value)
//   peekX     takes *parser, returns bool or string
//   consumeX  takes *parser and maybe a required literal, maybe returns value
//             also updates the parser position
//
// For nonterminals:
//   parseX    takes *parser, returns AST node, updates parser position

type parser struct {
	buf []byte
	pos int
}

func (p *parser) errorf(format string, args ...interface{}) {
	panic(parserErr{buf: p.buf, offset: p.pos, format: format, args: args})
}

// parse is the main entry point to the parser
func parse(buf []byte) (contract *contract, err error) {
	defer func() {
		if val := recover(); val != nil {
			if e, ok := val.(parserErr); ok {
				err = e
			} else {
				panic(val)
			}
		}
	}()
	p := &parser{buf: buf}
	c := parseContract(p)
	return
}

// parse functions

// contract name(p1, p2: t1, p3: t2) { ... }
func parseContract(p *parser) *contract {
	consumeKeyword(p, "contract")
	name := consumeIdentifier(p)
	params := parseParams(p)
	consumeTok(p, "{")
	clauses := parseClauses(p)
	consumeTok(p, "}")
	return &contract{name, params, clauses}
}

// (p1, p2: t1, p3: t2)
func parseParams(p *parser) []*param {
	var params []*param
	consumeTok(p, "(")
	first := true
	for !peekTok(p, ")") {
		if first {
			first = false
		} else {
			consumeTok(p, ",")
		}
		pt := parseParamsType(p)
		params = append(params, pt...)
	}
	consumeTok(p, ")")
	return params
}

func parseClauses(p *parser) []*clause {
	var clauses []*clause
	for !peekTok(p, "}") {
		c := parseClause(p)
		clauses = append(clauses, c)
	}
	return clauses
}

func parseParamsType(p *parser) []*param {
	firstName := consumeIdentifier(p)
	params := []*param{&param{name: firstName}}
	for peekTok(p, ",") {
		consumeTok(p, ",")
		name := consumeIdentifier(p)
		params = append(params, &param{name: name})
	}
	typ := consumeIdentifier(p)
	for _, p := range params {
		p.typ = typ
	}
	return params
}

func parseClause(p *parser) *clause {
	consumeKeyword(p, "clause")
	name := consumeIdentifier(p)
	params := parseParams(p)
	consumeTok(p, "{")
	statements := parseStatements(p)
	consumeTok(p, "}")
	return &clause{name: name, params: params, statements: statements}
}

func parseStatements(p *parser) []statement {
	var statements []statement
	for !peekTok(p, "}") {
		s := parseStatement(p)
		statements = append(statements, s)
	}
	return statements
}

func parseStatement(p *parser) statement {
	switch peekKeyword(p) {
	case "verify":
		return parseVerifyStmt(p)
	case "output":
		return parseOutputStmt(p)
	case "return":
		return parseReturnStmt(p)
	}
	panic(parseErr(p.buf, p.pos, "unknown keyword \"%s\"", peekKeyword(p)))
}

func parseVerifyStmt(p *parser) *verifyStatement {
	consumeKeyword(p, "verify")
	expr := parseExpr(p)
	return &verifyStatement{expr: expr}
}

func parseOutputStmt(p *parser) *outputStatement {
	consumeKeyword(p, "output")
	c := parseCallExpr(p)
	callExpr, ok := c.(*call)
	if !ok {
		p.errorf("expected call expression, got %T", c)
	}
	return &outputStatement{call: callExpr}
}

func parseReturnStmt(p *parser) *returnStatement {
	consumeKeyword(p, "return")
	expr := parseExpr(p)
	return &returnStatement{expr: expr}
}

func parseExpr(p *parser) expression {
	// Uses the precedence-climbing algorithm
	// <https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method>
	expr := parseUnaryExpr(p)
	expr2, pos := parseExprCont(p, expr, 0)
	if pos < 0 {
		p.errorf("expected expression")
	}
	p.pos = pos
	return expr2
}

func parseUnaryExpr(p *parser) expression {
	op, pos := scanUnaryOp(p.buf, p.pos)
	if pos < 0 {
		return parsePrimaryExpr(p)
	}
	p.pos = pos
	expr := parseUnaryExpr(p)
	return &unaryExpr{op: op, expr: expr}
}

func parseExprCont(p *parser, lhs expression, minPrecedence int) (expression, int) {
	for {
		opInfo, pos := scanBinaryOp(p.buf, p.pos)
		if pos < 0 || opInfo.precedence < minPrecedence {
			break
		}
		p.pos = pos

		rhs := parseUnaryExpr(p)

		for {
			opInfo2, pos2 := scanBinaryOp(p.buf, p.pos)
			if pos2 < 0 || opInfo2.precedence <= opInfo.precedence {
				break
			}
			rhs, p.pos = parseExprCont(p, rhs, opInfo2.precedence)
			if p.pos < 0 {
				return nil, -1 // or is this an error?
			}
		}
		lhs = &binaryExpr{left: lhs, right: rhs, op: opInfo.op}
	}
	return lhs, p.pos
}

func parsePrimaryExpr(p *parser) expression {
	if peekTok(p, "(") {
		consumeTok(p, "(")
		expr := parseExpr(p)
		consumeTok(p, ")")
		return expr
	}
	if expr, pos := scanLiteralExpr(p.buf, p.pos); pos >= 0 {
		p.pos = pos
		return expr
	}
	return parseCallExpr(p)
}

func parseCallExpr(p *parser) expression {
	name := consumeIdentifier(p) // xxx allow prop ref exprs too
	v := &varRef{name: name}
	if !peekTok(p, "(") {
		return v
	}
	args := parseArgs(p)
	return &call{fn: v, args: args}
}

func parseArgs(p *parser) []expression {
	var exprs []expression
	consumeTok(p, "(")
	first := true
	for !peekTok(p, ")") {
		if first {
			first = false
		} else {
			consumeTok(p, ",")
		}
		e := parseExpr(p)
		exprs = append(exprs, e)
	}
	consumeTok(p, ")")
	return exprs
}

// peek functions

func peekKeyword(p *parser) string {
	name, _ := scanIdentifier(p.buf, p.pos)
	return name
}

func peekTok(p *parser, token string) bool {
	pos := scanTok(p.buf, p.pos, token)
	return pos >= 0
}

// consume functions

func consumeKeyword(p *parser, keyword string) {
	pos := scanKeyword(p.buf, p.pos, keyword)
	if pos < 0 {
		p.errorf("expected keyword %s", keyword)
	}
	p.pos = pos
}

func consumeIdentifier(p *parser) string {
	name, pos := scanIdentifier(p.buf, p.pos)
	if pos < 0 {
		p.errorf("expected identifier")
	}
	p.pos = pos
	return name
}

func consumeTok(p *parser, token string) {
	pos := scanTok(p.buf, p.pos, token)
	if pos < 0 {
		p.errorf("expected %s token", token)
	}
	p.pos = pos
}

// scan functions

func scanUnaryOp(buf []byte, offset int) (*unaryOp, int) {
	// Maximum munch. Make sure "-3" scans as ("-3"), not ("-", "3").
	if _, pos := scanIntLiteral(buf, offset); pos >= 0 {
		return nil, -1
	}

	for _, op := range unaryOps {
		newOffset := scanTok(buf, offset, op.op)
		if newOffset >= 0 {
			return op, newOffset
		}
	}
	return nil, -1
}

func scanBinaryOp(buf []byte, offset int) (*binaryOp, int) {
	offset = skipWsAndComments(buf, offset)
	var (
		found     *binaryOp
		newOffset = -1
	)
	for _, op := range binaryOps {
		offset2 := scanTok(buf, offset, op.op)
		if offset2 >= 0 {
			if found == nil || len(op.op) > len(found.op) {
				found = op
				newOffset = offset2
			}
		}
	}
	return found, newOffset
}

func scanLiteralExpr(buf []byte, offset int) (*literal, int) {
	offset = skipWsAndComments(buf, offset)
	intliteral, newOffset := scanIntLiteral(buf, offset)
	if newOffset >= 0 {
		return intliteral, newOffset
	}
	strliteral, newOffset := scanStrLiteral(buf, offset)
	if newOffset >= 0 {
		return strliteral, newOffset
	}
	bytesliteral, newOffset := scanBytesLiteral(buf, offset) // 0x6c249a...
	if newOffset >= 0 {
		return bytesliteral, newOffset
	}
	return nil, -1
}

func scanIdentifier(buf []byte, offset int) (string, int) {
	offset = skipWsAndComments(buf, offset)
	i := offset
	for ; i < len(buf) && isIDChar(buf[i], i == offset); i++ {
	}
	if i == offset {
		return "", -1
	}
	return string(buf[offset:i]), i
}

func scanTok(buf []byte, offset int, s string) int {
	offset = skipWsAndComments(buf, offset)
	prefix := []byte(s)
	if bytes.HasPrefix(buf[offset:], prefix) {
		return offset + len(prefix)
	}
	return -1
}

func scanKeyword(buf []byte, offset int, keyword string) int {
	id, newOffset := scanIdentifier(buf, offset)
	if newOffset < 0 {
		return -1
	}
	if id != keyword {
		return -1
	}
	return newOffset
}

func scanIntLiteral(buf []byte, offset int) (*literal, int) {
	offset = skipWsAndComments(buf, offset)
	start := offset
	if offset < len(buf) && buf[offset] == '-' {
		offset++
	}
	i := offset
	for ; i < len(buf) && unicode.IsDigit(rune(buf[i])); i++ {
	}
	if i > offset {
		return newLiteral(buf[start:i], numType), i
	}
	return nil, -1
}

func scanStrLiteral(buf []byte, offset int) (*literal, int) {
	offset = skipWsAndComments(buf, offset)
	if offset >= len(buf) || buf[offset] != '\'' {
		return nil, -1
	}
	for i := offset + 1; i < len(buf); i++ {
		if buf[i] == '\'' {
			return newLiteral(buf[offset:i+1], bytesType), i + 1
		}
		if buf[i] == '\\' {
			i++
		}
	}
	panic(parseErr(buf, offset, "unterminated string literal"))
}

func scanBytesLiteral(buf []byte, offset int) (*literal, int) {
	offset = skipWsAndComments(buf, offset)
	if offset+4 >= len(buf) {
		return nil, -1
	}
	if buf[offset] != '0' || (buf[offset+1] != 'x' && buf[offset+1] != 'X') {
		return nil, -1
	}
	if !isHexDigit(buf[offset+2]) || !isHexDigit(buf[offset+3]) {
		return nil, -1
	}
	i := offset + 4
	for ; i < len(buf); i += 2 {
		if i == len(buf)-1 {
			panic(parseErr(buf, offset, "odd number of digits in hex literal"))
		}
		if !isHexDigit(buf[i]) {
			break
		}
		if !isHexDigit(buf[i+1]) {
			panic(parseErr(buf, offset, "odd number of digits in hex literal"))
		}
	}
	return newLiteral(buf[offset:i], bytesType), i
}

func skipWsAndComments(buf []byte, offset int) int {
	var inComment bool
	for ; offset < len(buf); offset++ {
		c := buf[offset]
		if inComment {
			if c == '\n' {
				inComment = false
			}
		} else {
			if c == '/' && offset < len(buf)-1 && buf[offset+1] == '/' {
				inComment = true
				offset++ // skip two chars instead of one
			} else if !unicode.IsSpace(rune(c)) {
				break
			}
		}
	}
	return offset
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isIDChar(c byte, initial bool) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c == '_' {
		return true
	}
	if initial {
		return false
	}
	return unicode.IsDigit(rune(c))
}

type parserErr struct {
	buf    []byte
	offset int
	format string
	args   []interface{}
}

func parseErr(buf []byte, offset int, format string, args ...interface{}) error {
	return parserErr{buf: buf, offset: offset, format: format, args: args}
}

func (p parserErr) Error() string {
	// Lines start at 1, columns start at 0, like nature intended.
	line := 1
	col := 0
	for i := 0; i < p.offset; i++ {
		if p.buf[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	args := []interface{}{line, col}
	args = append(args, p.args...)
	return fmt.Sprintf("line %d, col %d: "+p.format, args...)
}
