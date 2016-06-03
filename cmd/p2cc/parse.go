package main

import (
	"bytes"
	"fmt"
	"unicode"
)

// We have some function naming conventions.
//
// For terminals:
//   scanX     takes buf and position, returns new position (and maybe a value)
//   peekX     takes *parser, returns bool
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

func parse(buf []byte) (contracts []*contract, err error) {
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

func scanAssignOp(buf []byte, offset int) (string, int) {
	offset = skipWsAndComments(buf, offset)
	for _, op := range binaryOps {
		if op.canAssign {
			newOffset := scanTok(buf, offset, op.op)
			if newOffset >= 0 && newOffset < len(buf) && buf[newOffset] == '=' {
				return op.op + "=", newOffset + 1
			}
		}
	}
	if offset < len(buf) && buf[offset] == '=' && ((offset == len(buf)-1) || buf[offset+1] != '=') {
		return "=", offset + 1
	}
	return "", -1
}

func parseAssignStmt(p *parser) *assignStmt {
	id := consumeIdentifier(p)
	op, pos := scanAssignOp(p.buf, p.pos)
	if pos < 0 {
		p.errorf("expected assignment operator")
	}
	p.pos = pos
	expr := parseExpr(p)
	consumeTok(p, ";")
	return &assignStmt{name: id, expr: expr, op: op}
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

func parseBlock(p *parser, forClause bool) *block {
	consumeTok(p, "{")
	decls := parseDecls(p)
	stmts := parseStmts(p)
	block := &block{decls: decls, stmts: stmts}

	// TODO(kr): don't do this in the grammar,
	// check for it in a separate pass.
	if forClause && peekTok(p, "}") {
		if len(stmts) == 0 {
			p.errorf("empty clause body")
		}
		v, ok := stmts[len(stmts)-1].(*verifyStmt)
		if !ok {
			p.errorf("clause must end with expr or verify statement")
		}
		// The final statement of the clause block is a verify.  Convert
		// it to a bare expr so it's left on the stack after the
		// contract runs (saving two opcodes: VERIFY and TRUE [because
		// something true has to be left on the stack]).
		block.stmts = block.stmts[:len(stmts)-1]
		block.expr = v.expr
	} else if forClause {
		block.expr = parseExpr(p)
		if peekTok(p, ";") { // optional trailing semicolon
			consumeTok(p, ";")
		}
	}

	consumeTok(p, "}")
	return block
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

func parseCallExpr(p *parser) expr {
	name := consumeIdentifier(p)
	if !peekTok(p, "(") {
		return varref(name)
	}
	consumeTok(p, "(")

	var actuals []expr
	first := true
	for !peekTok(p, ")") {
		if !first {
			consumeTok(p, ",")
		}
		expr := parseExpr(p)
		actuals = append(actuals, expr)
		first = false
	}
	consumeTok(p, ")")
	var t int
	for _, c := range calls {
		if name == c.name {
			t = c.typ
			break
		}
	}
	return &callExpr{name: name, actuals: actuals, t: t}
}

func parseContract(p *parser) *contract {
	consumeKeyword(p, "contract")
	name := consumeIdentifier(p)
	params := parseParams(p)
	clauses := parseClauses(p)
	return &contract{name: name, params: params, clauses: clauses}
}

func parseContracts(p *parser) (contracts []*contract) {
	for peekKeyword(p) == "contract" {
		c := parseContract(p)
		contracts = append(contracts, c)
	}
	return contracts
}

func parseClause(p *parser) *clause {
	consumeKeyword(p, "clause")
	name := consumeIdentifier(p)
	params := parseParams(p)
	block := parseBlock(p, true)
	return &clause{name: name, params: params, block: block}
}

func parseClauses(p *parser) (clauses []*clause) {
	consumeTok(p, "{")
	for !peekTok(p, "}") {
		c := parseClause(p)
		clauses = append(clauses, c)
	}
	consumeTok(p, "}")
	return clauses
}

func parseDecl(p *parser) *decl {
	consumeKeyword(p, "var")
	id := consumeIdentifier(p)
	val := parseExpr(p)
	consumeTok(p, ";")
	return &decl{name: id, val: val}
}

func parseDecls(p *parser) (decls []*decl) {
	for peekKeyword(p) == "var" {
		decl := parseDecl(p)
		decls = append(decls, decl)
	}
	return decls
}

func parseExpr(p *parser) expr {
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

func parseExprCont(p *parser, lhs expr, minPrecedence int) (expr, int) {
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
		lhs = &binaryExpr{lhs: lhs, rhs: rhs, op: *op}
	}
	return lhs, p.pos
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

func consumeIdentifier(p *parser) string {
	name, pos := scanIdentifier(p.buf, p.pos)
	if pos < 0 {
		p.errorf("expected identifier")
	}
	p.pos = pos
	return name
}

func parseIfStmt(p *parser) *ifStmt {
	consumeKeyword(p, "if")
	condExpr := parseExpr(p)
	consequent := parseBlock(p, false)

	ifstmt := &ifStmt{condExpr: condExpr, consequent: consequent}
	if peekKeyword(p) == "else" {
		consumeKeyword(p, "else")
		ifstmt.alternate = parseBlock(p, false)
	}
	return ifstmt
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

func consumeKeyword(p *parser, keyword string) {
	pos := scanKeyword(p.buf, p.pos, keyword)
	if pos < 0 {
		p.errorf("expected keyword %s", keyword)
	}
	p.pos = pos
}

// Special case: returns the keyword rather than a bool,
// because it's just so darn useful.
func peekKeyword(p *parser) string {
	name, _ := scanIdentifier(p.buf, p.pos)
	return name
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

func parseParam(p *parser) (param typedName) {
	param.name = consumeIdentifier(p)
	if t, pos := scanTypeKeyword(p.buf, p.pos); pos >= 0 {
		p.pos = pos
		param.typ = t
	}
	return param
}

func parseParams(p *parser) (params []typedName) {
	consumeTok(p, "(")
	first := true
	for !peekTok(p, ")") {
		if !first {
			consumeTok(p, ",")
		}
		param := parseParam(p)
		params = append(params, param)
		first = false
	}
	consumeTok(p, ")")
	return params
}

func parsePrimaryExpr(p *parser) expr {
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

func parseStmt(p *parser) translatable {
	switch peekKeyword(p) {
	case "if":
		return parseIfStmt(p)
	case "while":
		return parseWhileStmt(p)
	case "verify":
		return parseVerifyStmt(p)
	}
	return parseAssignStmt(p)
}

func parseStmts(p *parser) (stmts []translatable) {
	peekStmt := func() bool {
		if kw := peekKeyword(p); kw == "if" || kw == "while" || kw == "verify" {
			return true
		}
		// no keyword, so check for assignment
		if _, pos := scanIdentifier(p.buf, p.pos); pos < 0 {
			return false
		}
		// (hack to look ahead by an extra token)
		p1 := new(parser)
		*p1 = *p
		consumeIdentifier(p1)
		_, pos := scanAssignOp(p1.buf, p1.pos)
		return pos >= 0
	}
	// TODO(kr): it would be better to check for '}' here,
	// but the grammar doesn't quite let us do that.
	for peekStmt() {
		stmt := parseStmt(p)
		stmts = append(stmts, stmt)
	}
	return stmts
}

func scanTok(buf []byte, offset int, s string) int {
	offset = skipWsAndComments(buf, offset)
	prefix := []byte(s)
	if bytes.HasPrefix(buf[offset:], prefix) {
		return offset + len(prefix)
	}
	return -1
}

func consumeTok(p *parser, token string) {
	pos := scanTok(p.buf, p.pos, token)
	if pos < 0 {
		p.errorf("expected %s token", token)
	}
	p.pos = pos
}

func peekTok(p *parser, token string) bool {
	pos := scanTok(p.buf, p.pos, token)
	return pos >= 0
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

func scanTypeKeyword(buf []byte, offset int) (int, int) {
	offset = skipWsAndComments(buf, offset)
	newOffset := scanKeyword(buf, offset, "num")
	if newOffset >= 0 {
		return numType, newOffset
	}
	newOffset = scanKeyword(buf, offset, "bool")
	if newOffset >= 0 {
		return boolType, newOffset
	}
	newOffset = scanKeyword(buf, offset, "bytes")
	if newOffset >= 0 {
		return bytesType, newOffset
	}
	return unknownType, -1
}

func parseUnaryExpr(p *parser) expr {
	op, pos := scanUnaryOp(p.buf, p.pos)
	if pos < 0 {
		return parsePrimaryExpr(p)
	}
	p.pos = pos
	expr := parseUnaryExpr(p)
	return &unaryExpr{expr: expr, op: *op}
}

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

func parseVerifyStmt(p *parser) *verifyStmt {
	consumeKeyword(p, "verify")
	expr := parseExpr(p)
	consumeTok(p, ";")
	return &verifyStmt{expr: expr}
}

func parseWhileStmt(p *parser) *whileStmt {
	consumeKeyword(p, "while")
	condExpr := parseExpr(p)
	body := parseBlock(p, false)
	whilestmt := &whileStmt{condExpr: condExpr, body: body}

	// Hark, a hack!  The translation of while <expr> { ...body... } is
	//   <expr> WHILE DROP ...body <expr> ENDWHILE
	// Repeating expr ensures it's evaluated once for each iteration of
	// the loop.  However, the translation of <expr> inside the WHILE
	// body may be different from the translation outside of it (because
	// the stack depth may be greater due to local var decls).  So we
	// overload the body.expr field, which is nominally for the optional
	// trailing expr in clause bodies, to contain a copy of the parsed
	// while condition and at translation time it will all Just Work.
	whilestmt.body.expr = whilestmt.condExpr

	return whilestmt
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
			if c == '#' {
				inComment = true
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
