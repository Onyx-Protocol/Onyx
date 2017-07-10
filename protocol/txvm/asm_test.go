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
	{`x"55"`, []byte{BaseData + 1, 0x55}},
	{`fail`, []byte{Fail}},
	// assemble only:
	{`{5, {6, 7}, 8}`, []byte{BaseInt + 8, BaseInt + 7, BaseInt + 6, BaseInt + 2, MakeTuple, BaseInt + 5, BaseInt + 3, MakeTuple}},
	{`{5, {6, 7}, [{2}]}`, []byte{BaseData + 3, BaseInt + 2, BaseInt + 1, MakeTuple, BaseInt + 7, BaseInt + 6, BaseInt + 2, MakeTuple, BaseInt + 5, BaseInt + 3, MakeTuple}},
	{`'test'`, []byte{BaseData + 4, 0x74, 0x65, 0x73, 0x74}},
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

func TestAssemble2(t *testing.T) {
	// A simple tx that sends 5 units of asset 0000... from one account to another
	src := `{'contract', {{5, x"0000000000000000000000000000000000000000000000000000000000000000"}}, {'program', ['txvm' inputstack inspect encode cat sha3 encode ['txvm' summarystack inspect encode cat sha3 1 datastack roll cat sha3 x"1111111111111111111111111111111111111111111111111111111111111111" checksig verify] cat 'program' 1 datastack roll 2 maketuple defer]}, x"2222222222222222222222222222222222222222222222222222222222222222"} unlock
{'program', ['txvm' inputstack inspect encode cat sha3 encode ['txvm' summarystack inspect encode cat sha3 1 datastack roll cat sha3 x"3333333333333333333333333333333333333333333333333333333333333333" checksig verify] cat 'program' 1 datastack roll 2 maketuple defer]} 1 lock
summarize
x"44444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444" satisfy`
	prog, err := Assemble(src)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%d bytes: %x", len(prog), prog)
	src2 := Disassemble(prog)
	t.Log(src2)
}

func TestDisassemble(t *testing.T) {
	for _, test := range asmValid[:len(asmValid)-3] {
		src := Disassemble(test.prog)
		if src != test.src {
			t.Errorf("Disassemble(%x) = %#q want %#q", test.prog, src, test.src)
		}
	}
}
