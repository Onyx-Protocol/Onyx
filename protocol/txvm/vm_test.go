package txvm

import (
	"testing"
)

func opTracer(t testing.TB) func(stack, byte, []byte, []byte) {
	return func(s stack, op byte, data, p []byte) {
		if op >= BaseData {
			t.Logf("[%x]\t\t#stack len: %d", data, s.Len())
		} else if op >= BaseInt {
			t.Logf("%d\t\t#stack len: %d", op-BaseInt, s.Len())
		} else {
			t.Logf("%s\t\t#stack len: %d", OpNames[op], s.Len())
		}
	}
}

func TestIssue(t *testing.T) {
	proof, err := Assemble(`
		10000 0 [1] 0 maketuple "6e6f6e6365"x 5 maketuple anchor
		100 [1] 0 maketuple "6173736574646566696e6974696f6e"x 3 maketuple issue
		[1] 1 ""x lock
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
		Out: [][32]byte{
			{
				0x17, 0x5f, 0x82, 0x98, 0x35, 0xf1, 0x95, 0x6a, 0xe3, 0x3d, 0x45, 0x23, 0x92, 0x65, 0xf1, 0x2c,
				0xf8, 0xed, 0x4f, 0x77, 0xda, 0x13, 0x9a, 0xa3, 0x23, 0xde, 0x20, 0x2b, 0xd7, 0x63, 0x80, 0xac,
			},
		},
		Nonce: [][32]byte{
			{
				0xff, 0x49, 0x64, 0x7c, 0x7d, 0xe2, 0xe1, 0x43, 0x41, 0xee, 0xe6, 0x7b, 0x7b, 0x57, 0x5f, 0x75,
				0xd0, 0x05, 0x58, 0x18, 0x1c, 0xbe, 0xa1, 0x45, 0x39, 0xf7, 0xa5, 0xac, 0x9d, 0x4a, 0x1a, 0xe5,
			},
		},
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestSpend(t *testing.T) {
	proof, err := Assemble(`
			[1 verify]
				"00112233445566778899aabbccddeeffffeeddccbbaa99887766554433221100"x
				100
				"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x
				0 maketuple
			"76616c7565"x 5 maketuple 1 maketuple
			"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"x
			0 maketuple
		"6f7574707574"x 5 maketuple unlock
		retire
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
		In: [][32]byte{
			{
				0x28, 0x3a, 0x23, 0x84, 0x0e, 0xb5, 0x78, 0x09, 0x0d, 0xce, 0xa9, 0x80, 0xf0, 0x82, 0xc3, 0x6a,
				0x2e, 0x4e, 0xcf, 0x4f, 0xc7, 0x1d, 0x2e, 0x00, 0x12, 0x6b, 0x6e, 0x23, 0xc9, 0x29, 0x20, 0xdc,
			},
		},
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestEntries(t *testing.T) {
	proof, err := Assemble(`
		10000 0 [1 verify] 0 maketuple "6e6f6e6365"x 5 maketuple anchor
		100 [1 verify] 0 maketuple "6173736574646566696e6974696f6e"x 3 maketuple issue
		45 split merge
		retire
		10000 0 ""x header
	`)
	if err != nil {
		t.Fatal(err)
	}
	tx := &Tx{
		Proof: proof,
		Nonce: [][32]byte{
			{
				0x99, 0x2b, 0xd1, 0x1e, 0xf2, 0xcc, 0xa8, 0xf6, 0xda, 0x5d, 0x2e, 0x4e, 0x03, 0x54, 0x09, 0xe8,
				0x8b, 0x64, 0xb4, 0xd9, 0xf3, 0x4e, 0x22, 0x72, 0x21, 0x59, 0x60, 0x67, 0xab, 0x27, 0xc1, 0x09,
			},
		},
	}
	ok := Validate(tx, TraceOp(opTracer(t)), TraceError(func(err error) { t.Error(err) }))
	if !ok {
		t.Fatal("expected ok")
	}
}
