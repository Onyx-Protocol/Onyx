package compiler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
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
func parse(buf []byte) (contracts []*Contract, err error) {
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
	contracts = parseContracts(p)
	return
}

// parse functions

func parseContracts(p *parser) []*Contract {
	var result []*Contract
	for peekKeyword(p) == "contract" {
		contract := parseContract(p)
		result = append(result, contract)
	}
	return result
}

// contract name(p1, p2: t1, p3: t2) locks value { ... }
func parseContract(p *parser) *Contract {
	consumeKeyword(p, "contract")
	name := consumeIdentifier(p)
	params := parseParams(p)
	consumeKeyword(p, "locks")
	value := consumeIdentifier(p)
	consumeTok(p, "{")
	clauses := parseClauses(p)
	consumeTok(p, "}")
	return &Contract{Name: name, Params: params, Clauses: clauses, Value: value}
}

// (p1, p2: t1, p3: t2)
func parseParams(p *parser) []*Param {
	var params []*Param
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

func parseClauses(p *parser) []*Clause {
	var clauses []*Clause
	for !peekTok(p, "}") {
		c := parseClause(p)
		clauses = append(clauses, c)
	}
	return clauses
}

func parseParamsType(p *parser) []*Param {
	firstName := consumeIdentifier(p)
	params := []*Param{&Param{Name: firstName}}
	for peekTok(p, ",") {
		consumeTok(p, ",")
		name := consumeIdentifier(p)
		params = append(params, &Param{Name: name})
	}
	consumeTok(p, ":")
	typ := consumeIdentifier(p)
	for _, parm := range params {
		if tdesc, ok := types[typ]; ok {
			parm.Type = tdesc
		} else {
			p.errorf("unknown type %s", typ)
		}
	}
	return params
}

func parseClause(p *parser) *Clause {
	var c Clause
	consumeKeyword(p, "clause")
	c.Name = consumeIdentifier(p)
	c.Params = parseParams(p)
	if peekKeyword(p) == "requires" {
		consumeKeyword(p, "requires")
		c.Reqs = parseClauseRequirements(p)
	}
	consumeTok(p, "{")
	c.statements = parseStatements(p)
	consumeTok(p, "}")
	return &c
}

func parseClauseRequirements(p *parser) []*ClauseReq {
	var result []*ClauseReq
	first := true
	for {
		switch {
		case first:
			first = false
		case peekTok(p, ","):
			consumeTok(p, ",")
		default:
			return result
		}
		var req ClauseReq
		req.Name = consumeIdentifier(p)
		consumeTok(p, ":")
		req.amountExpr = parseExpr(p)
		consumeKeyword(p, "of")
		req.assetExpr = parseExpr(p)
		result = append(result, &req)
	}
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
	case "lock":
		return parseLockStmt(p)
	case "unlock":
		return parseUnlockStmt(p)
	}
	panic(parseErr(p.buf, p.pos, "unknown keyword \"%s\"", peekKeyword(p)))
}

func parseVerifyStmt(p *parser) *verifyStatement {
	consumeKeyword(p, "verify")
	expr := parseExpr(p)
	return &verifyStatement{expr: expr}
}

func parseLockStmt(p *parser) *lockStatement {
	consumeKeyword(p, "lock")
	locked := parseExpr(p)
	consumeKeyword(p, "with")
	program := parseExpr(p)
	return &lockStatement{locked: locked, program: program}
}

func parseUnlockStmt(p *parser) *unlockStatement {
	consumeKeyword(p, "unlock")
	expr := parseExpr(p)
	return &unlockStatement{expr}
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
		return parseExpr2(p)
	}
	p.pos = pos
	expr := parseUnaryExpr(p)
	return &unaryExpr{op: op, expr: expr}
}

func parseExprCont(p *parser, lhs expression, minPrecedence int) (expression, int) {
	for {
		op, pos := scanBinaryOp(p.buf, p.pos)
		if pos < 0 || op.precedence < minPrecedence {
			break
		}
		p.pos = pos

		rhs := parseUnaryExpr(p)

		for {
			op2, pos2 := scanBinaryOp(p.buf, p.pos)
			if pos2 < 0 || op2.precedence <= op.precedence {
				break
			}
			rhs, p.pos = parseExprCont(p, rhs, op2.precedence)
			if p.pos < 0 {
				return nil, -1 // or is this an error?
			}
		}
		lhs = &binaryExpr{left: lhs, right: rhs, op: op}
	}
	return lhs, p.pos
}

func parseExpr2(p *parser) expression {
	if expr, pos := scanLiteralExpr(p.buf, p.pos); pos >= 0 {
		p.pos = pos
		return expr
	}
	return parseExpr3(p)
}

func parseExpr3(p *parser) expression {
	e := parseExpr4(p)
	if peekTok(p, "(") {
		args := parseArgs(p)
		return &callExpr{fn: e, args: args}
	}
	return e
}

func parseExpr4(p *parser) expression {
	if peekTok(p, "(") {
		consumeTok(p, "(")
		e := parseExpr(p)
		consumeTok(p, ")")
		return e
	}
	if peekTok(p, "[") {
		var elts []expression
		consumeTok(p, "[")
		first := true
		for !peekTok(p, "]") {
			if first {
				first = false
			} else {
				consumeTok(p, ",")
			}
			e := parseExpr(p)
			elts = append(elts, e)
		}
		consumeTok(p, "]")
		return listExpr(elts)
	}
	name := consumeIdentifier(p)
	return varRef(name)
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

var keywords = []string{
	"contract", "clause", "verify", "output", "return",
	"locks", "requires", "of", "lock", "with", "unlock",
}

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
			return &op, newOffset
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
	for i, op := range binaryOps {
		offset2 := scanTok(buf, offset, op.op)
		if offset2 >= 0 {
			if found == nil || len(op.op) > len(found.op) {
				found = &binaryOps[i]
				newOffset = offset2
			}
		}
	}
	return found, newOffset
}

// TODO(bobg): boolean literals?
func scanLiteralExpr(buf []byte, offset int) (expression, int) {
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

func scanIntLiteral(buf []byte, offset int) (integerLiteral, int) {
	offset = skipWsAndComments(buf, offset)
	start := offset
	if offset < len(buf) && buf[offset] == '-' {
		offset++
	}
	i := offset
	for ; i < len(buf) && unicode.IsDigit(rune(buf[i])); i++ {
	}
	if i > offset {
		n, err := strconv.ParseInt(string(buf[start:i]), 10, 64)
		if err != nil {
			return 0, -1
		}
		return integerLiteral(n), i
	}
	return 0, -1
}

func scanStrLiteral(buf []byte, offset int) (bytesLiteral, int) {
	offset = skipWsAndComments(buf, offset)
	if offset >= len(buf) || buf[offset] != '\'' {
		return bytesLiteral{}, -1
	}
	for i := offset + 1; i < len(buf); i++ {
		if buf[i] == '\'' {
			return bytesLiteral(buf[offset : i+1]), i + 1
		}
		if buf[i] == '\\' {
			i++
		}
	}
	panic(parseErr(buf, offset, "unterminated string literal"))
}

func scanBytesLiteral(buf []byte, offset int) (bytesLiteral, int) {
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
	decoded := make([]byte, hex.DecodedLen(i-(offset+2)))
	_, err := hex.Decode(decoded, buf[offset+2:i])
	if err != nil {
		return bytesLiteral{}, -1
	}
	return bytesLiteral(decoded), i
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
