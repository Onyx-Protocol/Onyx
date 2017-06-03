package vmutil

import (
	"bytes"
	"testing"

	"chain/protocol/txvm/op"
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
	{`16`, []byte{0x61, 16, op.Varint}},
	{`50`, []byte{0x61, 50, op.Varint}},
	{`0x50`, []byte{0x61, 0x50, op.Varint}},
	{`-1`, []byte{op.Neg1}},
	{`-2`, []byte{0x52, op.Neg1, op.Mul}},
	{`-15`, []byte{0x5f, op.Neg1, op.Mul}},
	{`-16`, []byte{0x61, 16, op.Varint, op.Neg1, op.Mul}},
	{`-9223372036854775808`, []byte{0x6a, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01, op.Varint}},
	{`"55"x`, []byte{0x61, 0x55}},
	{`input`, []byte{op.Input}},
	{`[input]`, []byte{0x61, op.Input, op.Prog}},
	{`[[input]]`, []byte{0x63, 0x61, op.Input, op.Prog, op.Prog}},
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
