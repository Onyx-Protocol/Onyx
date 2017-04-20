package name

import (
	"bytes"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

func Format(n pkix.Name) (string, error) {
	b := new(bytes.Buffer)
	seq := n.ToRDNSequence()
	for i := len(seq) - 1; i >= 0; i-- {
		err := writeRDN(b, seq[i])
		if err != nil {
			return "", err
		}
		if i > 0 {
			b.WriteByte(',')
		}
	}
	return b.String(), nil
}

func writeRDN(b *bytes.Buffer, rdn pkix.RelativeDistinguishedNameSET) error {
	// TODO(kr): sort?
	for i, atv := range rdn {
		if i > 0 {
			b.WriteByte('+')
		}
		useString := writeType(b, atv.Type)
		b.WriteByte('=')
		value, ok := atv.Value.(string)
		if !useString || !ok {
			b.WriteByte('#')
			der, err := asn1.Marshal(atv.Value)
			if err != nil {
				return err
			}
			buf := make([]byte, 2*len(der))
			hex.Encode(buf, der)
			b.Write(buf)
		} else {
			writeEscapedString(b, value)
		}
	}
	return nil
}

func writeType(b *bytes.Buffer, t asn1.ObjectIdentifier) (useString bool) {
	oid := t.String()
	if s, ok := oidName[oid]; ok {
		oid = s
		useString = true
	}
	b.WriteString(oid)
	return useString
}

func writeEscapedString(b *bytes.Buffer, s string) {
	for i, c := range s {
		if needEscape(i, len(s), c) {
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
}

func needEscape(i, n int, c rune) bool {
	return i == 0 && (c == ' ' || c == '#') ||
		i == n-1 && c == ' ' ||
		c < utf8.RuneSelf && strings.IndexByte("\x00\"+,;<>\\", byte(c)) >= 0
}
