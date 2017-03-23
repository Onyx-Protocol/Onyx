package vm

import (
	"bytes"
	"encoding/hex"
	"testing"

	"chain/errors"
)

func TestAssemble(t *testing.T) {
	cases := []struct {
		plain   string
		want    []byte
		wantErr error
	}{
		{"2 3 ADD 5 NUMEQUAL", mustDecodeHex("525393559c"), nil},
		{"0x02 3 ADD 5 NUMEQUAL", mustDecodeHex("01025393559c"), nil},
		{"19 14 SUB 5 NUMEQUAL", mustDecodeHex("01135e94559c"), nil},
		{"'Hello' 'WORLD' CAT 'HELLOWORLD' EQUAL", mustDecodeHex("0548656c6c6f05574f524c447e0a48454c4c4f574f524c4487"), nil},
		{`'H\'E' 'W' CAT 'H\'EW' EQUAL`, mustDecodeHex("0348274501577e044827455787"), nil},
		{`'HELLO '  'WORLD' CAT 'HELLO WORLD' EQUAL`, mustDecodeHex("0648454c4c4f2005574f524c447e0b48454c4c4f20574f524c4487"), nil},
		{`0x1`, nil, hex.ErrLength},
		{`BADTOKEN`, nil, ErrToken},
		{`'Unterminated quote`, nil, ErrToken},
	}

	for _, c := range cases {
		got, gotErr := Assemble(c.plain)

		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("Compile(%s) err = %v want %v", c.plain, errors.Root(gotErr), c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}

		if !bytes.Equal(got, c.want) {
			t.Errorf("Compile(%s) = %x want %x", c.plain, got, c.want)
		}
	}
}

func TestDisassemble(t *testing.T) {
	cases := []struct {
		raw     []byte
		want    string
		wantErr error
	}{
		{mustDecodeHex("525393559c"), "0x02 0x03 ADD 0x05 NUMEQUAL", nil},
		{mustDecodeHex("01135e94559c"), "0x13 0x0e SUB 0x05 NUMEQUAL", nil},
		{mustDecodeHex("6300000000"), "$alpha JUMP:$alpha", nil},
		{[]byte{0xff}, "NOPxff", nil},
	}

	for _, c := range cases {
		got, gotErr := Disassemble(c.raw)

		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("Decompile(%x) err = %v want %v", c.raw, errors.Root(gotErr), c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}

		if got != c.want {
			t.Errorf("Decompile(%x) = %s want %s", c.raw, got, c.want)
		}
	}
}

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}
