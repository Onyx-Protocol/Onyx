package txvm

import (
	"bytes"
	"encoding/hex"
	"testing"

	"chain/errors"
)

func TestAssembler(t *testing.T) {
	cases := []struct {
		src, wanthex string
		wanterr      error
	}{
		{"fail", "00", nil},
		{"pc", "01", nil},
		{"pc pc", "0101", nil},
		{"pushdata", "48", nil},
		{"0", "4e", nil},
		{"7", "55", nil},
		{"8", "48011047", nil},
		{"-1", "48010147", nil},
		{"bool", "1111", nil},
		{"1 dup 1", "4f4e4e074f", nil},
		{"x\"00010203\"", "480400010203", nil},
		{"'abcd'", "480461626364", nil},
		{"[fail]", "480100", nil},
		{"2 [1 dup 1] 2", "5048054f4e4e074f50", nil},
		{"{}", "4e0e", nil},
		{"{1, 2}", "504f500e", nil},
		{"{'abc', {5}, 'def'}", "4803646566534f0e4803616263510e", nil},
	}
	for i, c := range cases {
		b, err := Assemble(c.src)
		if err != nil {
			if c.wanterr == nil {
				t.Errorf("case %d: error: %s", i, err)
				continue
			}
			if errors.Root(c.wanterr) == errors.Root(err) {
				continue
			}
			t.Errorf("case %d: got error %s, want error %s", i, err, c.wanterr)
			continue
		}
		if c.wanterr != nil {
			t.Errorf("case %d: got no error, want error %s", i, c.wanterr)
			continue
		}
		wantbytes, _ := hex.DecodeString(c.wanthex)
		if !bytes.Equal(b, wantbytes) {
			t.Errorf("case %d: got %x, want %x", i, b, wantbytes)
		}
	}
}
