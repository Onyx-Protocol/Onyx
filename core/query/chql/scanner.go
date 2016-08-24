package chql

import "fmt"

type token int

const (
	tokInvalid token = iota
	tokEOF
	tokKeyword
	tokIdent
	tokString
	tokInteger
	tokPunct
	tokPlaceholder
)

func (t token) String() string {
	switch t {
	case tokInvalid:
		return "invalid"
	case tokEOF:
		return "EOF"
	case tokKeyword:
		return "keyword"
	case tokIdent:
		return "identifier"
	case tokString:
		return "string"
	case tokInteger:
		return "integer"
	case tokPunct:
		return "punctuation"
	case tokPlaceholder:
		return "placeholder"
	}
	return "unknown token"
}

// A scanner holds the scanner's internal state while processing
// a given text.
type scanner struct {
	// immutable state
	src []byte // source

	// scanning state
	ch     rune // current character
	offset int  // character offset
}

func (s *scanner) init(src []byte) {
	s.offset = -1
	s.src = src
	s.next() // advance onto the first input rune
}

// next reads the next ASCII char into s.ch.
// s.ch < 0 means end-of-file.
func (s *scanner) next() {
	if s.offset+1 < len(s.src) {
		s.offset++
		r := rune(s.src[s.offset])
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= 0x80:
			s.error(s.offset, "non-ASCII character")
		}
		s.ch = r
	} else {
		s.offset = len(s.src)
		s.ch = -1 // eof
	}
}

func (s *scanner) error(offs int, msg string) {
	panic(parseError{pos: offs, msg: msg})
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func (s *scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *scanner) scanMantissa(base int) {
	for digitVal(s.ch) < base {
		s.next()
	}
}

func (s *scanner) scanNumber() {
	// digitVal(s.ch) < 10
	if s.ch == '0' {
		// int
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
			}
		} else if digitVal(s.ch) < 10 {
			s.error(offs, "illegal leading 0 in number")
		}
	} else {
		// decimal int
		s.scanMantissa(10)
	}
}

func (s *scanner) scanString() {
	// "'" opening already consumed
	offs := s.offset - 1

	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '\'' {
			break
		}
		if ch == '\\' {
			s.error(offs, "illegal backslash in string literal")
		}
	}
}

func (s *scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' {
		s.next()
	}
}

func (s *scanner) Scan() (pos int, tok token, lit string) {
	s.skipWhitespace()

	// current token start
	pos = s.offset

	// determine token value
	switch ch := s.ch; {
	case isLetter(ch):
		lit = s.scanIdentifier()
		switch lit {
		case "AND", "OR":
			tok = tokKeyword
		default:
			tok = tokIdent
		}
		return pos, tok, lit
	case '0' <= ch && ch <= '9':
		s.scanNumber()
		tok = tokInteger
	default:
		s.next() // always make progress
		switch ch {
		case -1:
			return pos, tokEOF, ""
		case '\'':
			tok = tokString
			s.scanString()
		case '.', '(', ')', '=':
			tok = tokPunct
		case '$':
			s.scanMantissa(10)
			if s.offset-pos <= 1 {
				s.error(pos, "illegal $ character")
			}
			tok = tokPlaceholder
		default:
			s.error(pos, fmt.Sprintf("illegal character %q", ch))
		}
	}
	lit = string(s.src[pos:s.offset])
	return
}
