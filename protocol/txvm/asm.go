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
	progTok
	eofTok = -1
)

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
	var p []byte
	r := 0
	for r < len(src) {
		typ, lit, n := scan(src[r:])
		switch typ {
		case mnemonicTok:
			if opcode, ok := OpCodes[lit]; ok {
				p = append(p, opcode)
			} else if seq, ok := composite[lit]; ok {
				p = append(p, seq...)
			} else {
				return nil, errors.New("bad mnemonic " + lit)
			}
		case numberTok:
			v, _ := strconv.ParseInt(lit, 0, 64)
			if 0 <= v && v <= 0xf {
				p = append(p, BaseInt+byte(v))
			} else if v == -1 {
				p = append(p, BaseInt+1)
				p = append(p, Negate)
			} else if -0x10 < v && v < 0 {
				p = append(p, BaseInt+byte(-v))
				p = append(p, Negate)
			} else if v <= -0x10 && v != -v {
				p = append(p, pushData(encVarint(-v))...)
				p = append(p, Varint)
				p = append(p, Negate)
			} else {
				fmt.Println(v)
				fmt.Println(v != -v)
				p = append(p, pushData(encVarint(v))...)
				p = append(p, Varint)
			}
		case hexTok:
			s := lit[1 : len(lit)-2] // remove " and "x
			b, err := hex.DecodeString(s)
			if err != nil || lit[len(lit)-2] != '"' || lit[len(lit)-1] != 'x' {
				return nil, errors.New("bad hex string " + lit)
			}
			p = append(p, pushData(b)...)
		case progTok:
			if lit[len(lit)-1] != ']' {
				return nil, fmt.Errorf("parsing quoted program %s: missing ]", lit)
			}
			innerSrc := lit[1 : len(lit)-1]
			innerProg, err := Assemble(innerSrc)
			if err != nil {
				return nil, fmt.Errorf("parsing quoted program %s: %v", lit, err)
			}
			p = append(p, pushData(innerProg)...)
		default:
			return nil, errors.New("bad source")
		}
		r += n
	}
	return p, nil
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
		typ = progTok
		r = scanProg(src[n:])
	case c == '-' || isDigit(rune(c)):
		typ = numberTok
		r = scanNumber(src[n:])
	case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		typ = mnemonicTok
		r = scanWord(src[n:])
	}
	lit = src[n : n+r]
	n += r
	return
}

func skipWS(s string) (i int) {
	for ; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\n' {
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

func scanProg(s string) (i int) {
	for i < len(s) {
		c := s[i]
		i++
		if c == '[' {
			i += scanProg(s[i:])
		} else if c == ']' {
			return
		}
	}
	return i
}

func scanFunc(s string, f func(rune) bool) (n int) {
	for n < len(s) {
		c, r := utf8.DecodeRuneInString(s)
		if !f(c) {
			break
		}
		n += r
	}
	return n
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
			if len(instructions) > i+2 &&
				instructions[i+1].opcode == Varint &&
				instructions[i+2].opcode == Negate {
				v, _ := binary.Uvarint(inst.data)
				parts = append(parts, fmt.Sprintf("%d", -int64(v)))
				i += 2
			} else if len(instructions) > i+1 &&
				instructions[i+1].opcode == Varint {
				v, _ := binary.Uvarint(inst.data)
				parts = append(parts, fmt.Sprintf("%d", int64(v)))
				i += 1
			} else {
				parts = append(parts, fmt.Sprintf(`"%x"x`, inst.data))
			}
		} else if inst.opcode >= MinInt {
			if len(instructions) > i+1 &&
				instructions[i+1].opcode == Negate {
				parts = append(parts, fmt.Sprintf("%d", -(int(inst.opcode)-int(BaseInt))))
				i += 1
			} else {
				parts = append(parts, fmt.Sprintf("%d", int(inst.opcode)-int(BaseInt)))
			}
		} else {
			parts = append(parts, OpNames[inst.opcode])
		}
	}
	return strings.Join(parts, " ")
}
