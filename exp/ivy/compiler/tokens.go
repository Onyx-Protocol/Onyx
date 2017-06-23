package compiler

import (
	"fmt"
	"unicode"
)

const (
	tokEOF = 1 + iota
	tokLWSP
	tokNL
	tokComment
	tokKeyword
	tokIdentifier
	tokDelim // ( ) { } [ ] , :
	tokOp    // unary + binary ops
	tokIntLiteral
	tokStrLiteral
	tokBytesLiteral
)

type token struct {
	typ                  int
	line, column, offset int
	text                 []byte
}

var keywords = []string{
	"contract", "clause", "verify", "output", "return",
	"locks", "requires", "of", "lock", "with", "unlock",
}

func scan(inp []byte) ([]token, error) {
	line := 1
	column := 0
	offset := 0
	var res []token
	for offset < len(inp) {
		var (
			typ  int
			text []byte
			err  error
		)
		t := token{line: line, column: column, offset: offset}
		typ, line, column, offset, text, err = nextTok(inp, line, column, offset)
		if err != nil {
			return nil, err
		}
		t.typ = typ
		t.text = text
		res = append(res, t)
	}
	res = append(res, token{typ: tokEOF, line: line, column: column, offset: offset})
	return res, nil
}

func nextTok(inp []byte, line, column, offset int) (typOut, lineOut, columnOut, offsetOut int, text []byte, err error) {
	switch inp[offset] {
	case ' ', '\t':
		end := offset + 1
		for end < len(inp) && isLWSP(inp[end]) {
			end++
		}
		return tokLWSP, line, column + end - offset, end, inp[offset:end], nil

	case '\r':
		if offset == len(inp)-1 || inp[offset+1] != '\n' {
			err = fmt.Errorf("bare CR at position %d", offset)
			return
		}
		return tokNL, line + 1, 0, offset + 2, inp[offset : offset+2], nil

	case '\n':
		return tokNL, line + 1, 0, offset + 1, inp[offset : offset+1], nil

	case '/':
		if offset < len(inp)-1 && inp[offset+1] == '/' {
			end := offset + 2
			for end < len(inp) && inp[end] != '\r' && inp[end] != '\n' {
				end++
			}
			return tokComment, line, column + end - offset, end, inp[offset:end], nil
		}
		return tokOp, line, column + 1, offset + 1, inp[offset : offset+1], nil

	case '{', '}', '(', ')', '[', ']', ',', ':':
		return tokDelim, line, column + 1, offset + 1, inp[offset : offset+1], nil

	case '^', '|', '+', '&', '%', '*', '~':
		// unambiguous single-character operators
		return tokOp, line, column + 1, offset + 1, inp[offset : offset+1], nil

	case '<', '>':
		if offset < len(inp)-1 {
			switch inp[offset+1] {
			case inp[offset], '=':
				return tokOp, line, column + 2, offset + 2, inp[offset : offset+2], nil
			}
		}
		return tokOp, line, column + 1, offset + 1, inp[offset : offset+1], nil

	case '!', '=':
		// unary ! temporarily (?) disabled
		if offset == len(inp)-1 || inp[offset+1] != '=' {
			err = fmt.Errorf("unexpected character ! at position %d", offset)
			return
		}
		return tokOp, line, column + 2, offset + 2, inp[offset : offset+2], nil

	case '-':
		if offset < len(inp)-1 && unicode.IsDigit(rune(inp[offset+1])) {
			end := offset + 2
			for end < len(inp) && unicode.IsDigit(rune(inp[end])) {
				end++
			}
			return tokIntLiteral, line, column + end - offset, end, inp[offset:end], nil
		}
		return tokOp, line, column + 1, offset + 1, inp[offset : offset+1], nil

	case '0':
		if offset < len(inp)-1 && (inp[offset+1] == 'x' || inp[offset+1] == 'X') {
			if offset+4 > len(inp) {
				err = fmt.Errorf("incomplete bytes literal at position %d", offset)
				return
			}
			if !isHexDigit(inp[offset+2]) || !isHexDigit(inp[offset+3]) {
				err = fmt.Errorf("malformed bytes literal at position %d", offset)
				return
			}
			end := offset + 4
			for end < len(inp) && isHexDigit(inp[end]) {
				if end+2 > len(inp) {
					err = fmt.Errorf("incomplete bytes literal at position %d", offset)
					return
				}
				if !isHexDigit(inp[end+1]) {
					err = fmt.Errorf("malformed bytes literal at position %d", offset)
					return
				}
				end += 2
			}
			return tokBytesLiteral, line, column + end - offset, end, inp[offset:end], nil
		}
		fallthrough

	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		end := offset + 1
		for end < len(inp) && unicode.IsDigit(rune(inp[end])) {
			end++
		}
		return tokIntLiteral, line, column + end - offset, end, inp[offset:end], nil

	case '\'':
		end := offset + 1
		newLine := line
		newColumn := column + 1
		for end < len(inp) {
			if inp[end] == '\'' {
				return tokStrLiteral, newLine, newColumn + 1, end + 1, inp[offset : end+1], nil
			}
			if inp[end] == '\\' {
				newColumn++
				end++
				if end >= len(inp) {
					break
				}
			}
			if inp[end] == '\n' {
				newLine++
				newColumn = 0
			} else {
				newColumn++
			}
			end++
		}
		err = fmt.Errorf("unterminated string literal at position %d", offset)
		return

	default:
		if !isIDChar(inp[offset], true) {
			err = fmt.Errorf("unexpected character '%c' at position %d", inp[offset], offset)
			return
		}
		end := offset + 1
		for end < len(inp) && isIDChar(inp[end], false) {
			end++
		}
		typ := tokIdentifier
		id := string(inp[offset:end])
		for _, k := range keywords {
			if k == id {
				typ = tokKeyword
				break
			}
		}
		return typ, line, column + end - offset, end, inp[offset:end], nil
	}
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

func isLWSP(c byte) bool {
	return c == ' ' || c == '\t'
}

func tokConcat(tokens ...[]token) []token {
	var res []token
	for _, t := range tokens {
		res = append(res, t...)
	}
	return res
}
