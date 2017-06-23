package txvm

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	invalidTok = iota
	mnemonicTok
	numberTok
	hexTok
	progOpenTok
	progCloseTok
	tupleOpenTok
	tupleSepTok
	tupleCloseTok
	eofTok = -1
)

type token struct {
	typ int
	lit string
}

var composite = map[string][]byte{
	"bool":   {Not, Not},
	"verify": {PC, BaseInt + 4, Add, JumpIf, Fail},
	"jump":   {BaseInt + 1, BaseInt + 1, Roll, JumpIf},
	"max":    {Dup, BaseInt + 2, Roll, Dup, BaseInt + 2, Bury, GT, PC, BaseInt + 5, Add, JumpIf, BaseInt + 1, Roll, Drop},
	"min":    {Dup, BaseInt + 2, Roll, Dup, BaseInt + 2, Bury, GT, Not, PC, BaseInt + 5, Add, JumpIf, BaseInt + 1, Roll, Drop},
	"sub":    {BaseInt - 1, Mul, Add},
	"swap":   {BaseInt + 1, Roll},
}

// Notation:
//    word  mnemonic
//   12345  number
//   "aa"x  hex data
//   [dup]  quoted program
func Assemble(src string) ([]byte, error) {
	tokens := tokenize(src)
	return parse(tokens)
}

func parse(tokens []token) ([]byte, error) {
	var p []byte
	r := 0
	for r < len(tokens) {
		sub, n, err := parseStatement(tokens[r:])
		if err != nil {
			return nil, err
		}
		p = append(p, sub...)
		r += n
	}
	return p, nil
}

func parseStatement(tokens []token) ([]byte, int, error) {
	token := tokens[0]
	var p []byte
	switch token.typ {
	case mnemonicTok:
		if opcode, ok := OpCodes[token.lit]; ok {
			p = append(p, opcode)
		} else if seq, ok := composite[token.lit]; ok {
			p = append(p, seq...)
		} else {
			return nil, 0, errors.New("bad mnemonic " + token.lit)
		}
	case numberTok, hexTok, progOpenTok, tupleOpenTok:
		return parseValue(tokens)
	default:
		return nil, 0, fmt.Errorf("unexpected token: %s", token.lit)
	}
	return p, 1, nil
}

func parseValue(tokens []token) ([]byte, int, error) {
	token := tokens[0]
	switch token.typ {
	case numberTok:
		v, _ := strconv.ParseInt(token.lit, 0, 64)
		return pushInt64(v), 1, nil
	case hexTok:
		s := token.lit[1 : len(token.lit)-2] // remove " and "x
		b, err := hex.DecodeString(s)
		if err != nil || token.lit[len(token.lit)-2] != '"' || token.lit[len(token.lit)-1] != 'x' {
			return nil, 0, errors.New("bad hex string " + token.lit)
		}
		return pushData(b), 1, nil
	case progOpenTok:
		val, n, err := parseProgram(tokens[1:])
		if err != nil {
			return nil, 0, err
		}
		return val, n + 1, nil
	case tupleOpenTok:
		val, n, err := parseTuple(tokens[1:])
		if err != nil {
			return nil, 0, err
		}
		return val, n + 1, nil
	}
	return nil, 0, fmt.Errorf("unexpected token: %s", token.lit)
}

func parseProgram(tokens []token) ([]byte, int, error) {
	var p []byte
	r := 0
	for r < len(tokens) {
		token := tokens[r]
		switch token.typ {
		case progCloseTok:
			return pushData(p), r + 1, nil
		default:
			sub, n, err := parseStatement(tokens[r:])
			if err != nil {
				return nil, 0, err
			}
			p = append(p, sub...)
			r += n
		}
	}
	return nil, 0, errors.New("parsing quoted program missing ]")
}

func parseTuple(tokens []token) ([]byte, int, error) {
	var (
		vals        [][]byte
		r           = 0
		requiresSep = false
	)
	for r < len(tokens) {
		token := tokens[r]
		switch token.typ {
		case tupleCloseTok:
			var p []byte
			for i := len(vals) - 1; i >= 0; i-- {
				p = append(p, vals[i]...)
			}
			p = append(p, pushInt64(int64(len(vals)))...)
			p = append(p, MakeTuple)
			return p, r + 1, nil
		case tupleSepTok:
			if !requiresSep {
				return nil, 0, fmt.Errorf("unexpected token %s", token.lit)
			}
			requiresSep = false
			r++
		case numberTok, hexTok, progOpenTok, tupleOpenTok:
			if requiresSep {
				return nil, 0, errors.New("parsing tuple missing ,")
			}
			val, n, err := parseValue(tokens[r:])
			if err != nil {
				return nil, 0, err
			}
			vals = append(vals, val)
			requiresSep = true
			r += n
		default:
			return nil, 0, fmt.Errorf("unexpected token %s", token.lit)
		}
	}
	return nil, 0, errors.New("parsing tuple missing }")
}

func tokenize(src string) []token {
	r := 0
	var tokens []token
	for r < len(src) {
		typ, lit, n := scan(src[r:])
		tokens = append(tokens, token{typ: typ, lit: lit})
		r += n
	}
	return tokens
}

func scan(src string) (typ int, lit string, n int) {
	n = skipWS(src)
	r := 0
	if n >= len(src) {
		return eofTok, "", n
	}
	switch c := src[n]; {
	case c == '"':
		typ = hexTok
		r = scanString(src[n:])
	case c == '[':
		typ = progOpenTok
		r = 1
	case c == ']':
		typ = progCloseTok
		r = 1
	case c == '{':
		typ = tupleOpenTok
		r = 1
	case c == ',':
		typ = tupleSepTok
		r = 1
	case c == '}':
		typ = tupleCloseTok
		r = 1
	case c == '-' || isDigit(rune(c)):
		typ = numberTok
		r = scanNumber(src[n:])
	case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		typ = mnemonicTok
		r = scanWord(src[n:])
	}
	lit = src[n : n+r]
	n += r
	n += skipWS(src[n:])
	return
}

func skipWS(s string) (i int) {
	for ; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\n' || c == '\t' {
			continue
		}
		break
	}
	return i
}

func scanString(s string) int {
	n := 1 + scanFunc(s[1:], isHex) + 2
	if len(s) < n {
		return len(s)
	}
	return n
}

func scanNumber(s string) (n int) {
	if s[0] == '-' {
		n++
		s = s[1:]
	}
	return n + scanFunc(s, isDigit)
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isHex(r rune) bool {
	return isDigit(r) ||
		'a' <= r && r <= 'f' ||
		'A' <= r && r <= 'F'
}

func scanWord(s string) int {
	return scanFunc(s, unicode.IsLetter)
}

func scanFunc(s string, f func(rune) bool) (n int) {
	for n < len(s) {
		c, r := utf8.DecodeRuneInString(s[n:])
		if !f(c) {
			break
		}
		n += r
	}
	return n
}

func pushInt64(n int64) []byte {
	if 0 <= n && n <= 0xf {
		return []byte{BaseInt + byte(n)}
	} else if n <= -0x10 && n != -n {
		return append(pushData(encVarint(n)), Varint)
	} else {
		return append(pushData(encVarint(n)), Varint)
	}
}

func encVarint(v int64) []byte {
	buf := make([]byte, 10)
	buf = buf[:binary.PutUvarint(buf, uint64(v))]
	return buf
}

func pushData(buf []byte) (p []byte) {
	n := uint64(len(buf)) + BaseData
	pfx := make([]byte, 10)
	pfx = pfx[:binary.PutUvarint(pfx, n)]
	p = append(p, pfx...)
	p = append(p, buf...)
	return p
}

func Disassemble(prog []byte) string {
	pc := 0

	type instruction struct {
		opcode byte
		data   []byte
	}

	var instructions []instruction
	for pc < len(prog) {
		opcode, data, n := decodeInst(prog[pc:])
		pc += n
		instructions = append(instructions, instruction{opcode: opcode, data: data})
	}

	var parts []string
	for i := 0; i < len(instructions); i++ {
		inst := instructions[i]
		if inst.opcode >= BaseData {
			if len(instructions) > i+1 &&
				instructions[i+1].opcode == Varint {
				v, _ := binary.Uvarint(inst.data)
				parts = append(parts, fmt.Sprintf("%d", int64(v)))
				i += 1
			} else {
				parts = append(parts, fmt.Sprintf(`"%x"x`, inst.data))
			}
		} else if inst.opcode >= BaseInt {
			parts = append(parts, fmt.Sprintf("%d", int(inst.opcode)-int(BaseInt)))
		} else {
			parts = append(parts, OpNames[inst.opcode])
		}
	}
	return strings.Join(parts, " ")
}
