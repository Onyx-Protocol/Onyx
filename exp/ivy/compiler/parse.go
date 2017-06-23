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
	tokens []byte
	pos    int
}

func (p *parser) errorf(format string, args ...interface{}) {
	panic(parserErr{buf: p.buf, offset: p.pos, format: format, args: args})
}

// parse is the main entry point to the parser
func parse(buf []byte) (contracts []*Contract, err error) {
	tokens, err := scan(buf)
	if err != nil {
		return nil, err
	}

	defer func() {
		if val := recover(); val != nil {
			if e, ok := val.(parserErr); ok {
				err = e
			} else {
				panic(val)
			}
		}
	}()

	p := &parser{tokens: tokens}
	contracts = parseContracts(p)
	return
}

// parse functions

func parseContracts(p *parser) []*Contract {
	var result []*Contract
	for peekKeyword(p) == "contract" {
		contract, _ := parseContract(p)
		result = append(result, contract)
	}
	return result
}

// contract name(p1, p2: t1, p3: t2) locks value { ... }
func parseContract(p *parser) (*Contract, []token) {
	tok1 := consumeKeyword(p, "contract")
	name, tok2 := consumeIdentifier(p)
	params, tok3 := parseParams(p)
	tok4 := consumeKeyword(p, "locks")
	value, tok5 := consumeIdentifier(p)
	tok6 := consumeTok(p, "{")
	clauses, tok7 := parseClauses(p)
	tok8 := consumeTok(p, "}")
	tokens := tokConcat(tok1, tok2, tok3, tok4, tok5, tok6, tok7, tok8)
	return &Contract{Name: name, Params: params, Clauses: clauses, Value: value, tokens: tokens}, tokens
}

// (p1, p2: t1, p3: t2)
func parseParams(p *parser) ([]*Param, []token) {
	var params []*Param
	tokens := consumeTok(p, "(")
	first := true
	for !peekTok(p, ")") {
		if first {
			first = false
		} else {
			tok2 := consumeTok(p, ",")
			tokens = append(tokens, tok2)
		}
		pt, tok2 := parseParamsType(p)
		params = append(params, pt...)
		tokens = append(tokens, tok2...)
	}
	tok2 := consumeTok(p, ")")
	return params, tokConcat(tokens, tok2)
}

func parseClauses(p *parser) ([]*Clause, []token) {
	var (
		clauses []*Clause
		tokens  []token
	)
	for !peekTok(p, "}") {
		c, tok2 := parseClause(p)
		tokens = append(tokens, tok2...)
		clauses = append(clauses, c)
	}
	return clauses, tokens
}

func parseParamsType(p *parser) ([]*Param, []token) {
	firstName, tokens := consumeIdentifier(p)
	params := []*Param{&Param{Name: firstName, tokens: tokens}}
	for peekTok(p, ",") {
		tok2 := consumeTok(p, ",")
		name, tok3 := consumeIdentifier(p)
		params = append(params, &Param{Name: name, tokens: tok3})
		tokens = append(tokens, tokConcat(tok2, tok3)...)
	}
	tok4 := consumeTok(p, ":")
	typ, tok5 := consumeIdentifier(p)
	for _, parm := range params {
		if tdesc, ok := types[typ]; ok {
			parm.Type = tdesc
		} else {
			p.errorf("unknown type %s", typ)
		}
	}
	return params, tokConcat(tokens, tok4, tok5)
}

func parseClause(p *parser) (*Clause, []token) {
	var (
		c                                              Clause
		tok1, tok2, tok3, tok4, tok5, tok6, tok7, tok8 []token
	)
	tok1 = consumeKeyword(p, "clause")
	c.Name, tok2 = consumeIdentifier(p)
	c.Params, tok3 = parseParams(p)
	if peekKeyword(p) == "requires" {
		tok4 = consumeKeyword(p, "requires")
		c.Reqs, tok5 = parseClauseRequirements(p)
	}
	tok6 = consumeTok(p, "{")
	c.statements, tok7 = parseStatements(p)
	tok8 = consumeTok(p, "}")
	c.tokens = tokConcat(tok1, tok2, tok3, tok4, tok5, tok6, tok7, tok8)
	return &c, c.tokens
}

func parseClauseRequirements(p *parser) ([]*ClauseReq, []token) {
	var result []*ClauseReq
	first := true
	var tokens []token
	for {
		switch {
		case first:
			first = false
		case peekTok(p, ","):
			tok1 := consumeTok(p, ",")
			tokens = append(tokens, tok1...)
		default:
			return result, tokens
		}
		var (
			req                          ClauseReq
			tok2, tok3, tok4, tok5, tok6 []token
		)
		req.Name, tok2 = consumeIdentifier(p)
		tok3 = consumeTok(p, ":")
		req.amountExpr, tok4 = parseExpr(p)
		tok5 = consumeKeyword(p, "of")
		req.assetExpr, tok6 = parseExpr(p)
		req.tokens = tokConcat(tok2, tok3, tok4, tok5, tok6)
		tokens = append(tokens, req.tokens...)
		result = append(result, &req)
	}
}

func parseStatements(p *parser) ([]statement, []token) {
	var (
		statements []statement
		tokens     []token
	)
	for !peekTok(p, "}") {
		s, tok := parseStatement(p)
		statements = append(statements, s)
		tokens = append(tokens, tok...)
	}
	return statements, tokens
}

func parseStatement(p *parser) (statement, []token) {
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

func parseVerifyStmt(p *parser) (*verifyStatement, []token) {
	tok1 := consumeKeyword(p, "verify")
	expr, tok2 := parseExpr(p)
	tokens := tokConcat(tok1, tok2)
	return &verifyStatement{expr: expr, tokens: tokens}, tokens
}

func parseLockStmt(p *parser) (*lockStatement, []token) {
	tok1 := consumeKeyword(p, "lock")
	locked, tok2 := parseExpr(p)
	tok3 := consumeKeyword(p, "with")
	program, tok4 := parseExpr(p)
	tokens := tokConcat(tok1, tok2, tok3, tok4)
	return &lockStatement{locked: locked, program: program, tokens: tokens}, tokens
}

func parseUnlockStmt(p *parser) (*unlockStatement, []token) {
	tok1 := consumeKeyword(p, "unlock")
	expr, tok2 := parseExpr(p)
	tokens := tokConcat(tok1, tok2)
	return &unlockStatement{expr, tokens: tokens}, tokens
}

func parseExpr(p *parser) (expression, []token) {
	// Uses the precedence-climbing algorithm
	// <https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method>
	expr, tok1 := parseUnaryExpr(p)
	expr2, pos, tok2 := parseExprCont(p, expr, 0)
	if pos < 0 {
		p.errorf("expected expression")
	}
	p.pos = pos
	return expr2, tokConcat(tok1, tok2)
}

func parseUnaryExpr(p *parser) (expression, []token) {
	op, pos, tok1 := scanUnaryOp(p.buf, p.pos)
	if pos < 0 {
		return parseExpr2(p)
	}
	p.pos = pos
	expr, tok2 := parseUnaryExpr(p)
	tokens := tokConcat(tok1, tok2)
	return &unaryExpr{op: op, expr: expr, tokens: tokens}, tokens
}

func parseExprCont(p *parser, lhs expression, minPrecedence int) (expression, int, []token) {
	var tokens []token
	for {
		op, pos, tok1 := scanBinaryOp(p.buf, p.pos)
		if pos < 0 || op.precedence < minPrecedence {
			break
		}
		tokens = append(tokens, tok1...)
		p.pos = pos

		rhs, tok2 := parseUnaryExpr(p)
		tokens = append(tokens, tok2...)

		for {
			op2, pos2, _ := scanBinaryOp(p.buf, p.pos)
			if pos2 < 0 || op2.precedence <= op.precedence {
				break
			}
			var tok3 []token
			rhs, p.pos, tok3 = parseExprCont(p, rhs, op2.precedence)
			if p.pos < 0 {
				return nil, -1 // or is this an error?
			}
			tokens = append(tokens, tok3...)
		}
		lhs = &binaryExpr{left: lhs, right: rhs, op: op, tokens: tokens}
	}
	return lhs, p.pos, tokens
}

func parseExpr2(p *parser) (expression, []token) {
	if expr, pos, tok1 := scanLiteralExpr(p.buf, p.pos); pos >= 0 {
		p.pos = pos
		return expr, tok1
	}
	return parseExpr3(p)
}

func parseExpr3(p *parser) (expression, []token) {
	e, tok1 := parseExpr4(p)
	if peekTok(p, "(") {
		args, tok2 := parseArgs(p)
		tokens := tokConcat(tok1, tok2)
		return &callExpr{fn: e, args: args, tokens: tokens}, tokens
	}
	return e, tok1
}

func parseExpr4(p *parser) (expression, []token) {
	if peekTok(p, "(") {
		tok1 := consumeTok(p, "(")
		e, tok2 := parseExpr(p)
		tok3 := consumeTok(p, ")")
		return e, tokConcat(tok1, tok2, tok3)
	}
	if peekTok(p, "[") {
		var elts []expression
		tokens := consumeTok(p, "[")
		first := true
		for !peekTok(p, "]") {
			if first {
				first = false
			} else {
				tok2 := consumeTok(p, ",")
				tokens = append(tokens, tok2...)
			}
			e, tok3 := parseExpr(p)
			elts = append(elts, e)
			tokens = append(tokens, tok3...)
		}
		tok4 := consumeTok(p, "]")
		tokens = append(tokens, tok4...)
		return listExpr(elts), tokens // xxx make listExpr include tokens
	}
	name, tokens := consumeIdentifier(p)
	return varRef(name), tokens		// xxx make varRef include tokens
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
			return &op, newOffset
		}
	}
	return nil, -1
}

func scanBinaryOp(buf []byte, offset int) (*binaryOp, int) {
	offset, skipped = skipWsAndComments(buf, offset)
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
	return found, newOffset, skipped
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

func skipWsAndComments(buf []byte, offset int) (int, []byte) {
	var inComment bool
	startOffset := offset
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
	return offset, buf[startOffset:offset]
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
