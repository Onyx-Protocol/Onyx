package name

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type posError struct {
	s   string
	i   int64
	err error
}

func (p posError) Error() string {
	return fmt.Sprintf("pos %d: %v: parsing %q", p.i, p.err, p.s)
}

// Parse parses dn as an X.509 Distinguished Name.
// The supported fields are those of pkix.Name.
// Unsupported fields are silently discarded.
func Parse(dn string) (name pkix.Name, err error) {
	r := strings.NewReader(dn)
	defer func() {
		if rec := recover(); rec != nil {
			pos, _ := r.Seek(0, 1)
			err = posError{dn, pos, rec.(error)}
		}
	}()

	rdns := parseRDNs(r)
	name.FillFromRDNSequence(&rdns)
	return name, nil
}

func parseRDNs(r *strings.Reader) (rdns pkix.RDNSequence) {
	list(r, ',', func() {
		rdn := parseRDN(r)
		rdns = append(rdns, rdn)
	})
	return
}

func parseRDN(r *strings.Reader) (atvs pkix.RelativeDistinguishedNameSET) {
	list(r, '+', func() {
		atv := parseATV(r)
		atvs = append(atvs, atv)
	})
	return atvs
}

func parseATV(r *strings.Reader) (atv pkix.AttributeTypeAndValue) {
	atv.Type = parseAttributeType(r)
	requireAny(r, "=")
	atv.Value = parseAttributeValue(r)
	return atv
}

func parseAttributeType(r *strings.Reader) asn1.ObjectIdentifier {
	// attributeType = descr / numericoid

	// We first try descr, and if it fails,
	// seek back to where we were and try numericoid.
	restart, _ := r.Seek(0, 1)

	if s, ok := parseDescr(r); ok {
		oid, ok := oidNumber[s]
		if !ok {
			panic(fmt.Errorf("unknown descr %s", s))
		}
		r = strings.NewReader(oid)
	} else {
		r.Seek(restart, 0)
	}
	return parseNumOID(r)
}

func parseDescr(r *strings.Reader) (s string, ok bool) {
	// descr = leadkeychar *keychar
	// leadkeychar = ALPHA

	b := peek(r)
	ok = isAlpha(b)
	if !ok {
		return
	}
	for isKeychar(b) {
		s += string(b)
		r.ReadByte()
		b = peek(r)
	}
	return
}

func parseNumOID(r *strings.Reader) (oid asn1.ObjectIdentifier) {
	// numericoid = number 1*( DOT number )
	list(r, '.', func() {
		n := parseNumber(r)
		oid = append(oid, n)
	})
	if len(oid) < 2 {
		panic(fmt.Errorf("%q (expected '.')", peek(r)))
	}
	return
}

func parseNumber(r *strings.Reader) int {
	// number  = DIGIT / ( LDIGIT 1*DIGIT )
	// DIGIT   = %x30 / LDIGIT       ; "0"-"9"
	// LDIGIT  = %x31-39             ; "1"-"9"

	b := peek(r)
	if b == '0' {
		r.ReadByte()
		return 0
	}
	if !('1' <= b && b <= '9') {
		panic(fmt.Errorf("%q (expected 0-9)", b))
	}
	s := string(capture(r, func() { scan(r, isDigit) }))
	n, _ := strconv.Atoi(s)
	return n
}

func parseAttributeValue(r *strings.Reader) interface{} {
	// attributeValue = string / hexstring
	if peek(r) == '#' {
		return parseHexstring(r)
	}
	return parseString(r)
}

func parseString(r *strings.Reader) string {
	// string =   [ ( leadchar / pair ) [ *( stringchar / pair )
	//    ( trailchar / pair ) ] ]

	// Note, we decode multi-byte UTF-8 sequences
	// one byte at a time.
	// The only code points that require special treatment (escaping)
	// are in ASCII,
	// so we simply pass through all non-ASCII bytes.
	// (We can check for valid UTF8 at the eng
	// using a library routine.)

	// TODO(kr): maybe use a regexp for this

	b := peek(r)
	if !isLeadByte(b) && b != '\\' {
		panic(fmt.Errorf("%q (expected string)", b))
	}
	endsWithEsc := false
	esc := capture(r, func() {
		b = peek(r)
		for isStringByte(b) || b == '\\' {
			if b == '\\' {
				endsWithEsc = true
				scanEscSeq(r)
				b = peek(r)
				continue
			}
			endsWithEsc = false
			r.ReadByte()
			b = peek(r)
		}
	})
	if b := esc[len(esc)-1]; !isTrailByte(b) && !endsWithEsc {
		panic(fmt.Errorf("%q (expected trail char or escape sequence)", b))
	}
	return unescaper.Replace(string(esc)) // TODO(kr): unescape
}

func scanEscSeq(r *strings.Reader) {
	// pair = ESC ( ESC / special / hexpair )
	// special = escaped / SPACE / SHARP / EQUALS
	// escaped = DQUOTE / PLUS / COMMA / SEMI / LANGLE / RANGLE

	requireAny(r, `\`)
	const pairSet = `\"+,;<> #=`
	if b := peek(r); strings.IndexByte(pairSet, b) >= 0 {
		r.ReadByte()
		return
	}
	const hexChars = "0123456789abcdefABCDEF"
	requireAny(r, hexChars)
	requireAny(r, hexChars)
}

func parseHexstring(r *strings.Reader) (v interface{}) {
	// hexstring = SHARP 1*hexpair
	// hexpair = HEX HEX
	requireAny(r, "#")
	if !isHex(peek(r)) {
		panic(errors.New("expected hexstring"))
	}
	h := capture(r, func() { scan(r, isHex) })
	b := make([]byte, len(h)/2)
	_, err := hex.Decode(b, h)
	if err != nil {
		panic(err)
	}
	rest, err := asn1.Unmarshal(b, &v)
	if err != nil {
		panic(err)
	} else if len(rest) > 0 {
		panic(errors.New("trailing garbage"))
	}
	return v
}

func isAlpha(b byte) bool {
	return 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z'
}

func isKeychar(b byte) bool {
	// keychar = ALPHA / DIGIT / HYPHEN
	return isAlpha(b) || isDigit(b) || b == '-'
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func isHex(b byte) bool {
	return isDigit(b) || 'a' <= b && b <= 'f' || 'A' <= b && b <= 'F'
}

func isStringByte(b byte) bool {
	// stringchar = SUTF1 / UTFMB
	// SUTF1 = %x01-21 / %x23-2A / %x2D-3A /
	//    %x3D / %x3F-5B / %x5D-7F
	return 0x01 <= b && b <= 0x21 ||
		0x23 <= b && b <= 0x2a ||
		0x2d <= b && b <= 0x3a ||
		0x3d == b ||
		0x3f <= b && b <= 0x5b ||
		0x5d <= b && b <= 0x7f ||
		b >= utf8.RuneSelf
}

func isLeadByte(b byte) bool {
	// leadchar = LUTF1 / UTFMB
	// LUTF1 = %x01-1F / %x21 / %x24-2A / %x2D-3A /
	//    %x3D / %x3F-5B / %x5D-7F
	return 0x01 <= b && b <= 0x1f ||
		0x21 == b ||
		0x24 <= b && b <= 0x2a ||
		0x2d <= b && b <= 0x3a ||
		0x3d == b ||
		0x3f <= b && b <= 0x5b ||
		0x5d <= b && b <= 0x7f ||
		b >= utf8.RuneSelf

}

func isTrailByte(b byte) bool {
	// trailchar  = TUTF1 / UTFMB
	// TUTF1 = %x01-1F / %x21 / %x23-2A / %x2D-3A /
	// 	  %x3D / %x3F-5B / %x5D-7F
	return 0x01 <= b && b <= 0x1f ||
		0x21 == b ||
		0x23 <= b && b <= 0x2a ||
		0x2d <= b && b <= 0x3a ||
		0x3d == b ||
		0x3f <= b && b <= 0x5b ||
		0x5d <= b && b <= 0x7f ||
		b >= utf8.RuneSelf
}

func list(r *strings.Reader, c byte, f func()) {
	for {
		f()
		if peek(r) != c {
			return
		}
		requireAny(r, string(c))
	}
}

func scan(r *strings.Reader, f func(byte) bool) {
	c := peek(r)
	for f(c) {
		r.ReadByte()
		c = peek(r)
	}
}

func capture(r *strings.Reader, f func()) []byte {
	beg, _ := r.Seek(0, 1)
	f()
	end, _ := r.Seek(0, 1)
	b := make([]byte, end-beg)
	r.ReadAt(b, beg)
	return b
}

func peek(r *strings.Reader) byte {
	b, err := r.ReadByte()
	if err != nil {
		return 0
	}
	r.UnreadByte()
	return b
}

func requireAny(r *strings.Reader, set string) byte {
	b, err := r.ReadByte()
	if err != nil {
		panic(err)
	}
	if strings.IndexByte(set, b) < 0 {
		panic(fmt.Errorf("%q (expected %q)", b, set))
	}
	return b
}
