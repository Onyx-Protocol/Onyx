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
	{`1`, []byte{BaseInt + 1}},
	{`10`, []byte{BaseInt + 10}},
	{`11`, []byte{BaseInt + 11}},
	{`14`, []byte{BaseInt + 14}},
	{`16`, []byte{BaseData + 1, 16, Varint}},
	{`50`, []byte{BaseData + 1, 50, Varint}},
	{`-1`, []byte{BaseData + 10, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, Varint}},
	{`-2`, []byte{BaseData + 10, 0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, Varint}},
	{`-14`, []byte{BaseData + 10, 0xf2, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, Varint}},
	{`-16`, []byte{BaseData + 10, 0xf0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, Varint}},
	{`-9223372036854775808`, []byte{BaseData + 10, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01, Varint}},
	{`"55"x`, []byte{BaseData + 1, 0x55}},
	{`fail`, []byte{Fail}},
	// assemble only:
	{`{5, {6, 7}, 8}`, []byte{BaseInt + 8, BaseInt + 7, BaseInt + 6, BaseInt + 2, MakeTuple, BaseInt + 5, BaseInt + 3, MakeTuple}},
	{`{5, {6, 7}, [{2}]}`, []byte{BaseData + 3, BaseInt + 2, BaseInt + 1, MakeTuple, BaseInt + 7, BaseInt + 6, BaseInt + 2, MakeTuple, BaseInt + 5, BaseInt + 3, MakeTuple}},
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

func TestDisassemble(t *testing.T) {
	for _, test := range asmValid[:len(asmValid)-2] {
		src := Disassemble(test.prog)
		if src != test.src {
			t.Errorf("Disassemble(%x) = %#q want %#q", test.prog, src, test.src)
		}
	}
}
