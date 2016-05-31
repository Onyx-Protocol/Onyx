package main

import (
	"bytes"
	"fmt"
	"unicode"
)

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

	contracts, _ = parseContracts(buf, 0)
	return
}

func parseAssignOp(buf []byte, offset int) (string, int) {
	offset = skipWsAndComments(buf, offset)
	for _, op := range binaryOps {
		if op.canAssign {
			newOffset := parseStr(buf, offset, op.op)
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

func parseAssignStmt(buf []byte, offset int) (*assignStmt, int) {
	id, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	op, newOffset := parseAssignOp(buf, offset)
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	expr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected rhs expr for assignment"))
	}
	offset = newOffset
	newOffset = parseStr(buf, offset, ";")
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected semicolon"))
	}
	return &assignStmt{name: id, expr: expr, op: op}, newOffset
}

func parseBinaryOp(buf []byte, offset int) (*binaryOp, int) {
	offset = skipWsAndComments(buf, offset)
	var (
		found     *binaryOp
		newOffset = -1
	)
	for _, op := range binaryOps {
		offset2 := parseStr(buf, offset, op.op)
		if offset2 >= 0 {
			if found == nil || len(op.op) > len(found.op) {
				found = op
				newOffset = offset2
			}
		}
	}
	return found, newOffset
}

func parseBlock(buf []byte, offset int, forClause bool) (*block, int) {
	newOffset := parseStr(buf, offset, "{")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	decls, newOffset := parseDecls(buf, offset)
	if newOffset >= 0 {
		offset = newOffset
	}
	stmts, newOffset := parseStmts(buf, offset)
	if newOffset >= 0 {
		offset = newOffset
	}
	block := &block{decls: decls, stmts: stmts}
	if forClause {
		block.expr, newOffset = parseExpr(buf, offset)
		if newOffset >= 0 {
			newOffset2 := parseStr(buf, newOffset, ";") // optional trailing semicolon
			if newOffset2 >= 0 {
				offset = newOffset2
			} else {
				offset = newOffset
			}
		} else {
			if len(stmts) == 0 {
				panic(parseErr(buf, offset, "empty clause body"))
			}
			v, ok := stmts[len(stmts)-1].(*verifyStmt)
			if !ok {
				panic(parseErr(buf, offset, "clause must end with expr or verify statement"))
			}
			// The final statement of the clause block is a verify.  Convert
			// it to a bare expr so it's left on the stack after the
			// contract runs (saving two opcodes: VERIFY and TRUE [because
			// something true has to be left on the stack]).
			block.stmts = block.stmts[:len(stmts)-1]
			block.expr = v.expr
		}
	}
	newOffset = parseStr(buf, offset, "}")
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected close brace"))
	}
	return block, newOffset
}

func parseBytesLiteral(buf []byte, offset int) (*literal, int) {
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

func parseCallExpr(buf []byte, offset int) (*callExpr, int) {
	name, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	newOffset = parseStr(buf, offset, "(")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	var actuals []expr
	first := true
	for {
		newOffset = parseStr(buf, offset, ")")
		if newOffset >= 0 {
			var t int
			for _, c := range calls {
				if name == c.name {
					t = c.typ
					break
				}
			}
			return &callExpr{name: name, actuals: actuals, t: t}, newOffset
		}
		if !first {
			newOffset = parseStr(buf, offset, ",")
			if newOffset < 0 {
				panic(parseErr(buf, offset, "expected comma in parameter list"))
			}
			offset = newOffset
		}
		expr, newOffset := parseExpr(buf, offset)
		if newOffset < 0 {
			panic(parseErr(buf, offset, "expected actual argument"))
		}
		offset = newOffset
		actuals = append(actuals, expr)
		first = false
	}
}

func parseContract(buf []byte, offset int) (*contract, int) {
	newOffset := parseKeyword(buf, offset, "contract")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	name, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected identifier after 'contract'"))
	}
	offset = newOffset
	params, newOffset := parseParams(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected contract parameter list"))
	}
	offset = newOffset
	clauses, newOffset := parseClauses(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected contract clauses"))
	}
	return &contract{name: name, params: params, clauses: clauses}, newOffset
}

func parseContracts(buf []byte, offset int) ([]*contract, int) {
	var contracts []*contract
	for {
		contract, newOffset := parseContract(buf, offset)
		if newOffset < 0 {
			return contracts, offset
		}
		contracts = append(contracts, contract)
		offset = newOffset
	}
}

func parseClause(buf []byte, offset int) (*clause, int) {
	newOffset := parseKeyword(buf, offset, "clause")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	name, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected clause name"))
	}
	offset = newOffset
	params, newOffset := parseParams(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected clause params"))
	}
	offset = newOffset
	block, newOffset := parseBlock(buf, offset, true)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected clause body"))
	}
	return &clause{name: name, params: params, block: block}, newOffset
}

func parseClauses(buf []byte, offset int) ([]*clause, int) {
	newOffset := parseStr(buf, offset, "{")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	var clauses []*clause
	for {
		newOffset = parseStr(buf, offset, "}")
		if newOffset >= 0 {
			return clauses, newOffset
		}
		clause, newOffset := parseClause(buf, offset)
		if newOffset < 0 {
			panic(parseErr(buf, offset, "expected a clause"))
		}
		clauses = append(clauses, clause)
		offset = newOffset
	}
}

func parseDecl(buf []byte, offset int) (*decl, int) {
	newOffset := parseKeyword(buf, offset, "var")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	id, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected var name"))
	}
	offset = newOffset
	val, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected initializer expression"))
	}
	offset = newOffset
	newOffset = parseStr(buf, offset, ";")
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected semicolon"))
	}
	return &decl{name: id, val: val}, newOffset
}

func parseDecls(buf []byte, offset int) ([]*decl, int) {
	var decls []*decl
	for {
		decl, newOffset := parseDecl(buf, offset)
		if newOffset < 0 {
			return decls, offset
		}
		offset = newOffset
		decls = append(decls, decl)
	}
}

func parseExpr(buf []byte, offset int) (expr, int) {
	// Uses the precedence-climbing algorithm
	// <https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method>
	expr, newOffset := parsePrimaryExpr(buf, offset)
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	expr2, newOffset := parseExprCont(buf, offset, expr, 0)
	if newOffset < 0 {
		return nil, -1
	}
	return expr2, newOffset
}

func parseExprCont(buf []byte, offset int, lhs expr, minPrecedence int) (expr, int) {
	for {
		op, offset2 := parseBinaryOp(buf, offset)
		if offset2 < 0 || op.precedence < minPrecedence {
			break
		}
		offset = offset2

		var rhs expr
		rhs, offset2 = parsePrimaryExpr(buf, offset)
		if offset2 < 0 {
			panic(parseErr(buf, offset, "expected rhs expr after binary operator %s", op.op))
		}
		offset = offset2

		for {
			op2, offset3 := parseBinaryOp(buf, offset)
			if offset3 < 0 || op2.precedence <= op.precedence {
				break
			}
			rhs, offset = parseExprCont(buf, offset, rhs, op2.precedence)
			if offset < 0 {
				return nil, -1 // or is this an error?
			}
		}
		lhs = &binaryExpr{lhs: lhs, rhs: rhs, op: *op}
	}
	return lhs, offset
}

func parseIdentifier(buf []byte, offset int) (string, int) {
	offset = skipWsAndComments(buf, offset)
	i := offset
	for ; i < len(buf) && isIDChar(buf[i], i == offset); i++ {
	}
	if i == offset {
		return "", -1
	}
	return string(buf[offset:i]), i
}

func parseIfStmt(buf []byte, offset int) (*ifStmt, int) {
	newOffset := parseKeyword(buf, offset, "if")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	condExpr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected 'if' condition"))
	}
	offset = newOffset
	consequent, newOffset := parseBlock(buf, offset, false)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected 'if' body"))
	}
	offset = newOffset
	ifstmt := &ifStmt{condExpr: condExpr, consequent: consequent}
	newOffset = parseKeyword(buf, offset, "else")
	if newOffset >= 0 {
		ifstmt.alternate, newOffset = parseBlock(buf, newOffset, false)
		if newOffset < 0 {
			panic(parseErr(buf, offset, "expected 'else' body"))
		}
		offset = newOffset
	}
	return ifstmt, offset
}

func parseIntLiteral(buf []byte, offset int) (*literal, int) {
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

func parseKeyword(buf []byte, offset int, keyword string) int {
	id, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		return -1
	}
	if id != keyword {
		return -1
	}
	return newOffset
}

func parseLiteralExpr(buf []byte, offset int) (*literal, int) {
	offset = skipWsAndComments(buf, offset)
	intliteral, newOffset := parseIntLiteral(buf, offset)
	if newOffset >= 0 {
		return intliteral, newOffset
	}
	strliteral, newOffset := parseStrLiteral(buf, offset)
	if newOffset >= 0 {
		return strliteral, newOffset
	}
	bytesliteral, newOffset := parseBytesLiteral(buf, offset) // 0x6c249a...
	if newOffset >= 0 {
		return bytesliteral, newOffset
	}
	return nil, -1
}

func parseParam(buf []byte, offset int) (typedName, int) {
	var param typedName
	id, newOffset := parseIdentifier(buf, offset)
	if newOffset < 0 {
		return param, -1
	}
	offset = newOffset
	param.name = id
	t, newOffset := parseTypeKeyword(buf, offset)
	if newOffset >= 0 {
		param.typ = t
		offset = newOffset
	}
	return param, offset
}

func parseParams(buf []byte, offset int) ([]typedName, int) {
	newOffset := parseStr(buf, offset, "(")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	var params []typedName
	first := true
	for {
		newOffset = parseStr(buf, offset, ")")
		if newOffset >= 0 {
			return params, newOffset
		}
		if !first {
			newOffset = parseStr(buf, offset, ",")
			if newOffset < 0 {
				panic(parseErr(buf, offset, "expected comma in parameter list"))
			}
			offset = newOffset
		}
		param, newOffset := parseParam(buf, offset)
		if newOffset < 0 {
			panic(parseErr(buf, offset, "expected parameter"))
		}
		params = append(params, param)
		offset = newOffset
		first = false
	}
}

func parseParenExpr(buf []byte, offset int) (expr, int) {
	newOffset := parseStr(buf, offset, "(")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	expr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		return nil, -1 // or is this an error?
	}
	offset = newOffset
	newOffset = parseStr(buf, offset, ")")
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected close paren"))
	}
	return expr, newOffset
}

func parsePrimaryExpr(buf []byte, offset int) (expr, int) {
	offset = skipWsAndComments(buf, offset)

	parenexpr, newOffset := parseParenExpr(buf, offset)
	if newOffset >= 0 {
		return parenexpr, newOffset
	}
	unaryexpr, newOffset := parseUnaryExpr(buf, offset)
	if newOffset >= 0 {
		return unaryexpr, newOffset
	}
	callexpr, newOffset := parseCallExpr(buf, offset)
	if newOffset >= 0 {
		return callexpr, newOffset
	}
	varrefexpr, newOffset := parseVarRefExpr(buf, offset)
	if newOffset >= 0 {
		return varrefexpr, newOffset
	}
	literalexpr, newOffset := parseLiteralExpr(buf, offset)
	if newOffset >= 0 {
		return literalexpr, newOffset
	}
	return nil, -1
}

func parseStmt(buf []byte, offset int) (translatable, int) {
	offset = skipWsAndComments(buf, offset)
	ifstmt, newOffset := parseIfStmt(buf, offset)
	if newOffset >= 0 {
		return ifstmt, newOffset
	}
	whilestmt, newOffset := parseWhileStmt(buf, offset)
	if newOffset >= 0 {
		return whilestmt, newOffset
	}
	verifystmt, newOffset := parseVerifyStmt(buf, offset)
	if newOffset >= 0 {
		return verifystmt, newOffset
	}
	assignstmt, newOffset := parseAssignStmt(buf, offset)
	if newOffset >= 0 {
		return assignstmt, newOffset
	}
	return nil, -1
}

func parseStmts(buf []byte, offset int) ([]translatable, int) {
	var stmts []translatable
	for {
		stmt, newOffset := parseStmt(buf, offset)
		if newOffset < 0 {
			return stmts, offset
		}
		offset = newOffset
		stmts = append(stmts, stmt)
	}
}

func parseStr(buf []byte, offset int, s string) int {
	offset = skipWsAndComments(buf, offset)
	prefix := []byte(s)
	if bytes.HasPrefix(buf[offset:], prefix) {
		return offset + len(prefix)
	}
	return -1
}

func parseStrLiteral(buf []byte, offset int) (*literal, int) {
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

func parseTypeKeyword(buf []byte, offset int) (int, int) {
	offset = skipWsAndComments(buf, offset)
	newOffset := parseKeyword(buf, offset, "num")
	if newOffset >= 0 {
		return numType, newOffset
	}
	newOffset = parseKeyword(buf, offset, "bool")
	if newOffset >= 0 {
		return boolType, newOffset
	}
	newOffset = parseKeyword(buf, offset, "bytes")
	if newOffset >= 0 {
		return bytesType, newOffset
	}
	return unknownType, -1
}

func parseUnaryExpr(buf []byte, offset int) (*unaryExpr, int) {
	op, newOffset := parseUnaryOp(buf, offset)
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	expr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		return nil, -1 // or is this an error?
	}
	return &unaryExpr{expr: expr, op: *op}, newOffset
}

func parseUnaryOp(buf []byte, offset int) (*unaryOp, int) {
	for _, op := range unaryOps {
		newOffset := parseStr(buf, offset, op.op)
		if newOffset >= 0 {
			return op, newOffset
		}
	}
	return nil, -1
}

func parseVarRefExpr(buf []byte, offset int) (*varref, int) {
	name, newOffset := parseIdentifier(buf, offset)
	if newOffset >= 0 {
		v := varref(name)
		return &v, newOffset
	}
	return nil, -1
}

func parseVerifyStmt(buf []byte, offset int) (*verifyStmt, int) {
	newOffset := parseKeyword(buf, offset, "verify")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	expr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected expression for 'verify'"))
	}
	offset = newOffset
	newOffset = parseStr(buf, offset, ";")
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected semicolon"))
	}
	return &verifyStmt{expr: expr}, newOffset
}

func parseWhileStmt(buf []byte, offset int) (*whileStmt, int) {
	newOffset := parseKeyword(buf, offset, "while")
	if newOffset < 0 {
		return nil, -1
	}
	offset = newOffset
	condExpr, newOffset := parseExpr(buf, offset)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected 'while' condition"))
	}
	offset = newOffset
	body, newOffset := parseBlock(buf, offset, false)
	if newOffset < 0 {
		panic(parseErr(buf, offset, "expected 'while' body"))
	}

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

	return whilestmt, newOffset
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
