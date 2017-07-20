package txvm

import (
	"bytes"
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
	stringTok
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

var composite = map[string]string{
	"bool":   "not not",
	"dup":    "0 datastack peek",
	"jump":   "1 swap jumpif",
	"max":    "dup 2 datastack roll dup 2 datastack bury greaterthan pc 5 add jumpif swap drop",
	"min":    "dup 2 datastack roll dup 2 datastack bury greaterthan not pc 5 add jumpif swap drop",
	"swap":   "1 datastack roll",
	"verify": "pc 4 add jumpif fail",
}

var stackNames = map[string]int64{
	"datastack":    datastack,
	"altstack":     altstack,
	"entrystack":   entrystack,
	"commandstack": commandstack,
	"effectstack":  effectstack,
}

// Notation:
//    word     mnemonic
//   12345     number
//   x"aa"     hex data
//   'foo'     string
//   [dup]     quoted program
//   {x, y, z} tuple (encoded as "push z, push y, push x, push 3, 'tuple'")
func Assemble(src string) ([]byte, error) {
	tokens := tokenize(src)
	return parse(tokens)
}

func Disassemble(prog []byte) (string, error) {
	var result []string
	for pc := int64(0); pc < int64(len(prog)); {
		opcode := prog[pc]
		pc++
		switch {
		case isSmallIntOp(opcode):
			result = append(result, fmt.Sprintf("%d", opcode-MinSmallInt))
		case int(opcode) >= len(opNames):
			result = append(result, fmt.Sprintf("nop%d", opcode))
		case opcode == OpPushdata:
			data, n, err := decodePushdata(prog[pc:])
			if err != nil {
				return "", err
			}
			pc += n
			var s string
			switch {
			case pc < int64(len(prog)) && prog[pc] == OpInt64:
				num, n2 := binary.Varint(data)
				if n2 != len(data) {
					return "", fmt.Errorf("wrong length for encoded varint (%d vs. %d)", n2, len(data))
				}
				s = fmt.Sprintf("%d", num)
			case strings.IndexFunc(string(data), isUnprintable) < 0:
				s = fmt.Sprintf("'%s'", strEscape(data))
			default:
				s = fmt.Sprintf("x\"%x\"", data)
			}
			result = append(result, s)
		default:
			if name := opNames[opcode]; name != "" {
				result = append(result, name)
			} else {
				result = append(result, fmt.Sprintf("nop%d", opcode))
			}
		}
	}
	// TODO: map op patterns to "composite" abbrevs
	// TODO: map "... tuple" to "{...}"
	// TODO: map some strings to disassembled program literals
	return strings.Join(result, " "), nil
}

func isUnprintable(r rune) bool {
	return !unicode.IsPrint(r)
}

func strEscape(in []byte) string {
	b := new(bytes.Buffer)
	for _, c := range in {
		switch c {
		case '\\', '\'':
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return string(b.Bytes())
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
		if opcode, ok := opCodes[token.lit]; ok {
			return append(p, opcode), 1, nil
		}
		if seq, ok := composite[token.lit]; ok {
			p2, err := parse(tokenize(seq))
			if err != nil {
				return nil, 0, err
			}
			return append(p, p2...), 1, nil
		}
		if stackNum, ok := stackNames[token.lit]; ok {
			return append(p, pushInt64(stackNum)...), 1, nil
		}
		return nil, 0, errors.New("bad mnemonic " + token.lit)

	case numberTok, hexTok, stringTok, progOpenTok, tupleOpenTok:
		return parseValue(tokens)

	default:
		return nil, 0, fmt.Errorf("unexpected token: %s", token.lit)
	}
}

func parseValue(tokens []token) ([]byte, int, error) {
	token := tokens[0]
	switch token.typ {
	case numberTok:
		v, _ := strconv.ParseInt(token.lit, 0, 64)
		return pushInt64(v), 1, nil
	case hexTok:
		s := token.lit[2 : len(token.lit)-1] // remove x" and "
		b, err := hex.DecodeString(s)
		if err != nil || token.lit[0] != 'x' || token.lit[1] != '"' || token.lit[len(token.lit)-1] != '"' {
			return nil, 0, errors.New("bad hex literal " + token.lit)
		}
		return encodePushdata(b), 1, nil
	case stringTok:
		s := token.lit[1 : len(token.lit)-1]
		if token.lit[len(token.lit)-1] != '\'' {
			return nil, 0, errors.New("bad text literal " + token.lit)
		}
		return encodePushdata([]byte(s)), 1, nil
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
			return encodePushdata(p), r + 1, nil
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
			p = append(p, OpTuple)
			return p, r + 1, nil
		case tupleSepTok:
			if !requiresSep {
				return nil, 0, fmt.Errorf("unexpected token %s", token.lit)
			}
			requiresSep = false
			r++
		case numberTok, hexTok, stringTok, progOpenTok, tupleOpenTok:
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
	case c == 'x':
		typ = hexTok
		r = scanHex(src[n:])
	case c == '\'':
		typ = stringTok
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
	default:
		typ = invalidTok
		r = 1
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
	n := 1
	for n < len(s) {
		if s[n] == '\'' {
			return n + 1
		}
		n++
	}
	return n
}

func scanHex(s string) int {
	n := 3 + scanFunc(s[2:], isHex)
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

func isAlphaNum(r rune) bool {
	return isDigit(r) || unicode.IsLetter(r)
}

func scanWord(s string) int {
	return scanFunc(s, isAlphaNum)
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

func pushInt64(num int64) []byte {
	if isSmallInt(num) {
		return []byte{MinSmallInt + byte(num)}
	}
	var buf [10]byte
	n := binary.PutVarint(buf[:], num)
	return append(encodePushdata(buf[:n]), OpInt64)
}
