package compiler

import (
	"fmt"
	"strconv"
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
	tokens []token
	pos    int
}

func (p *parser) errorf(format string, args ...interface{}) {
	panic(parserErr{tokens: p.tokens, pos: p.pos, format: format, args: args})
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
	tok6 := consumeDelim(p, "{")
	clauses, tok7 := parseClauses(p)
	tok8 := consumeDelim(p, "}")
	tokens := tokConcat(tok1, tok2, tok3, tok4, tok5, tok6, tok7, tok8)
	return &Contract{Name: name, Params: params, Clauses: clauses, Value: value, tokens: tokens}, tokens
}

// (p1, p2: t1, p3: t2)
func parseParams(p *parser) ([]*Param, []token) {
	var params []*Param
	tokens := consumeDelim(p, "(")
	first := true
	for !peekDelim(p, ")") {
		if first {
			first = false
		} else {
			tok2 := consumeDelim(p, ",")
			tokens = append(tokens, tok2...)
		}
		pt, tok3 := parseParamsType(p)
		params = append(params, pt...)
		tokens = append(tokens, tok3...)
	}
	tok4 := consumeDelim(p, ")")
	return params, tokConcat(tokens, tok4)
}

func parseClauses(p *parser) ([]*Clause, []token) {
	var (
		clauses []*Clause
		tokens  []token
	)
	for !peekDelim(p, "}") {
		c, tok2 := parseClause(p)
		tokens = append(tokens, tok2...)
		clauses = append(clauses, c)
	}
	return clauses, tokens
}

func parseParamsType(p *parser) ([]*Param, []token) {
	firstName, tokens := consumeIdentifier(p)
	params := []*Param{&Param{Name: firstName, tokens: tokens}}
	for peekDelim(p, ",") {
		tok2 := consumeDelim(p, ",")
		name, tok3 := consumeIdentifier(p)
		params = append(params, &Param{Name: name, tokens: tok3})
		tokens = append(tokens, tokConcat(tok2, tok3)...)
	}
	tok4 := consumeDelim(p, ":")
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
	tok6 = consumeDelim(p, "{")
	c.statements, tok7 = parseStatements(p)
	tok8 = consumeDelim(p, "}")
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
		case peekDelim(p, ","):
			tok1 := consumeDelim(p, ",")
			tokens = append(tokens, tok1...)
		default:
			return result, tokens
		}
		var (
			req                          ClauseReq
			tok2, tok3, tok4, tok5, tok6 []token
		)
		req.Name, tok2 = consumeIdentifier(p)
		tok3 = consumeDelim(p, ":")
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
	for !peekDelim(p, "}") {
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
	panic(parseErr(p.tokens, p.pos, "unknown keyword \"%s\"", peekKeyword(p)))
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
	return &unlockStatement{expr: expr, tokens: tokens}, tokens
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
	op, pos, tok1 := scanUnaryOp(p.tokens, p.pos)
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
		op, pos, tok1 := scanBinaryOp(p.tokens, p.pos)
		if pos < 0 || op.precedence < minPrecedence {
			break
		}
		tokens = append(tokens, tok1...)
		p.pos = pos

		rhs, tok2 := parseUnaryExpr(p)
		tokens = append(tokens, tok2...)

		for {
			op2, pos2, _ := scanBinaryOp(p.tokens, p.pos)
			if pos2 < 0 || op2.precedence <= op.precedence {
				break
			}
			var tok3 []token
			rhs, p.pos, tok3 = parseExprCont(p, rhs, op2.precedence)
			if p.pos < 0 {
				return nil, -1, nil // or is this an error?
			}
			tokens = append(tokens, tok3...)
		}
		lhs = &binaryExpr{left: lhs, right: rhs, op: op, tokens: tokens}
	}
	return lhs, p.pos, tokens
}

func parseExpr2(p *parser) (expression, []token) {
	if expr, pos, tok1 := scanLiteralExpr(p.tokens, p.pos); pos >= 0 {
		p.pos = pos
		return expr, tok1
	}
	return parseExpr3(p)
}

func parseExpr3(p *parser) (expression, []token) {
	e, tok1 := parseExpr4(p)
	if peekDelim(p, "(") {
		args, tok2 := parseArgs(p)
		tokens := tokConcat(tok1, tok2)
		return &callExpr{fn: e, args: args, tokens: tokens}, tokens
	}
	return e, tok1
}

func parseExpr4(p *parser) (expression, []token) {
	if peekDelim(p, "(") {
		tok1 := consumeDelim(p, "(")
		e, tok2 := parseExpr(p)
		tok3 := consumeDelim(p, ")")
		return e, tokConcat(tok1, tok2, tok3)
	}
	if peekDelim(p, "[") {
		var elts []expression
		tokens := consumeDelim(p, "[")
		first := true
		for !peekDelim(p, "]") {
			if first {
				first = false
			} else {
				tok2 := consumeDelim(p, ",")
				tokens = append(tokens, tok2...)
			}
			e, tok3 := parseExpr(p)
			elts = append(elts, e)
			tokens = append(tokens, tok3...)
		}
		tok4 := consumeDelim(p, "]")
		tokens = append(tokens, tok4...)
		return listExpr(elts), tokens // xxx make listExpr include tokens
	}
	name, tokens := consumeIdentifier(p)
	return varRef(name), tokens // xxx make varRef include tokens
}

func parseArgs(p *parser) ([]expression, []token) {
	var exprs []expression
	tokens := consumeDelim(p, "(")
	first := true
	for !peekDelim(p, ")") {
		if first {
			first = false
		} else {
			tok2 := consumeDelim(p, ",")
			tokens = append(tokens, tok2...)
		}
		e, tok3 := parseExpr(p)
		exprs = append(exprs, e)
		tokens = append(tokens, tok3...)
	}
	tok4 := consumeDelim(p, ")")
	tokens = append(tokens, tok4...)
	return exprs, tokens
}

// peek functions

func peekKeyword(p *parser) string {
	pos, _ := skipWsAndComments(p.tokens, p.pos)
	if pos >= len(p.tokens) {
		return ""
	}
	tok := p.tokens[pos]
	if tok.typ != tokKeyword {
		return ""
	}
	return string(tok.text)
}

func peekDelim(p *parser, token string) bool {
	pos, _ := scanDelim(p.tokens, p.pos, token)
	return pos >= 0
}

// consume functions

func consumeKeyword(p *parser, keyword string) []token {
	pos, tokens := scanKeyword(p.tokens, p.pos, keyword)
	if pos < 0 {
		p.errorf("expected keyword %s", keyword)
	}
	p.pos = pos
	return tokens
}

func consumeIdentifier(p *parser) (string, []token) {
	name, pos, tokens := scanIdentifier(p.tokens, p.pos)
	if pos < 0 {
		p.errorf("expected identifier")
	}
	p.pos = pos
	return name, tokens
}

func consumeDelim(p *parser, token string) []token {
	pos, tokens := scanDelim(p.tokens, p.pos, token)
	if pos < 0 {
		p.errorf("expected %s token", token)
	}
	p.pos = pos
	return tokens
}

// scan functions

func skipWsAndComments(tokens []token, pos int) (int, []token) {
	end := pos
loop:
	for end < len(tokens) {
		switch tokens[end].typ {
		case tokLWSP, tokNL, tokComment:
			end++
		default:
			break loop
		}
	}
	return end, tokens[pos:end]
}

func scanUnaryOp(tokens []token, pos int) (*unaryOp, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return nil, -1, nil
	}
	tok := tokens[pos]
	if tok.typ == tokOp {
		for _, op := range unaryOps {
			if op.op == string(tok.text) {
				return &op, pos + 1, append(skipped, tok)
			}
		}
	}
	return nil, -1, nil
}

func scanBinaryOp(tokens []token, pos int) (*binaryOp, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return nil, -1, nil
	}
	tok := tokens[pos]
	if tok.typ == tokOp {
		for _, op := range binaryOps {
			if op.op == string(tok.text) {
				return &op, pos + 1, append(skipped, tok)
			}
		}
	}
	return nil, -1, nil
}

// TODO(bobg): boolean literals?
func scanLiteralExpr(tokens []token, pos int) (expression, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	intliteral, newPos, tok := scanIntLiteral(tokens, pos)
	if newPos >= 0 {
		return intliteral, newPos, tokConcat(skipped, tok)
	}
	strliteral, newPos, tok := scanStrLiteral(tokens, pos)
	if newPos >= 0 {
		return strliteral, newPos, tokConcat(skipped, tok)
	}
	bytesliteral, newPos, tok := scanBytesLiteral(tokens, pos) // 0x6c249a...
	if newPos >= 0 {
		return bytesliteral, newPos, tokConcat(skipped, tok)
	}
	return nil, -1, nil
}

func scanIdentifier(tokens []token, pos int) (string, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return "", -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokIdentifier {
		return "", -1, nil
	}
	return string(tok.text), pos + 1, append(skipped, tok)
}

func scanDelim(tokens []token, pos int, s string) (int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokDelim {
		return -1, nil
	}
	if string(tok.text) != s {
		return -1, nil
	}
	return pos + 1, append(skipped, tok)
}

func scanKeyword(tokens []token, pos int, keyword string) (int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokKeyword {
		return -1, nil
	}
	if string(tok.text) != keyword {
		return -1, nil
	}
	return pos + 1, append(skipped, tok)
}

func scanIntLiteral(tokens []token, pos int) (integerLiteral, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return 0, -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokIntLiteral {
		return 0, -1, nil
	}
	n, err := strconv.ParseInt(string(tok.text), 10, 64)
	if err != nil {
		return 0, -1, nil
	}
	return integerLiteral(n), pos + 1, append(skipped, tok)
}

func scanStrLiteral(tokens []token, pos int) (bytesLiteral, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return bytesLiteral{}, -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokStrLiteral {
		return bytesLiteral{}, -1, nil
	}
	var (
		escape    bool
		unescaped []byte
	)
	for _, b := range tok.text[1 : len(tok.text)-1] {
		if escape {
			unescaped = append(unescaped, b)
			escape = false
		} else if b == '\\' {
			escape = true
		} else {
			unescaped = append(unescaped, b)
		}
	}
	return bytesLiteral(unescaped), pos + 1, append(skipped, tok)
}

func scanBytesLiteral(tokens []token, pos int) (bytesLiteral, int, []token) {
	var skipped []token
	pos, skipped = skipWsAndComments(tokens, pos)
	if pos >= len(tokens) {
		return bytesLiteral{}, -1, nil
	}
	tok := tokens[pos]
	if tok.typ != tokBytesLiteral {
		return bytesLiteral{}, -1, nil
	}
	return bytesLiteral(tok.text), pos + 1, append(skipped, tok)
}

type parserErr struct {
	tokens []token
	pos    int
	format string
	args   []interface{}
}

func parseErr(tokens []token, pos int, format string, args ...interface{}) error {
	return parserErr{tokens: tokens, pos: pos, format: format, args: args}
}

func (p parserErr) Error() string {
	pos := p.pos
	if pos > len(p.tokens) {
		pos = len(p.tokens) - 1
	}
	tok := p.tokens[pos]
	args := []interface{}{tok.line, tok.column}
	args = append(args, p.args...)
	return fmt.Sprintf("line %d, col %d: "+p.format, args...)
}
