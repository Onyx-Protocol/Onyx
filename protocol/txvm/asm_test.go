package txvm

import (
	"bytes"
	"testing"
)

var asmValid = []struct {
	src  string
	prog []byte
}{
	{``, []byte{}},
	{`1`, []byte{0x51}},
	{`10`, []byte{0x5a}},
	{`11`, []byte{0x5b}},
	{`15`, []byte{0x5f}},
	{`16`, []byte{BaseData + 1, 16, Varint}},
	{`50`, []byte{BaseData + 1, 50, Varint}},
	{`0x50`, []byte{BaseData + 1, 0x50, Varint}},
	{`-1`, []byte{MinInt}},
	{`-2`, []byte{0x52, MinInt, Mul}},
	{`-15`, []byte{0x5f, MinInt, Mul}},
	{`-16`, []byte{BaseData + 1, 16, Varint, MinInt, Mul}},
	{`-9223372036854775808`, []byte{BaseData + 10, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01, Varint}},
	{`"55"x`, []byte{BaseData + 1, 0x55}},
	{`fail`, []byte{Fail}},
}

func TestAssemble(t *testing.T) {
	for _, test := range asmValid {
		prog, err := Assemble(test.src)
		if err != nil {
			t.Errorf("Assemble(%#q) err = %v want nil", test.src, err)
			continue
		}
		if !bytes.Equal(prog, test.prog) {
			t.Errorf("Assemble(%#q) = %x want %x", test.src, prog, test.prog)
		}
	}
}

/*
func TestDisassemble(t *testing.T) {
	for _, test := range asmValid {
		src, err := Disassemble(test.prog)
		if err != nil {
			t.Errorf("Disassemble(%x) err = %v want nil", test.prog, err)
			continue
		}
		if src != test.src {
			t.Errorf("Disassemble(%x) = %#q want %#q", test.prog, src, test.src)
		}
	}
}
*/
