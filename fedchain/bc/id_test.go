package bc

import "testing"

var idTests = []struct {
	encoded string
	decoded [32]byte
}{
	{
		"0000000000000000000000000000000000000000000000000000000000000000",
		[32]byte{},
	},
	{
		"0000000000000000000000000000000000000000000000000000000000000001",
		[32]byte{1},
	},
	{
		"0000000000000000000000000000000000000000000000000000000000000201",
		[32]byte{1, 2},
	},
}

func TestID(t *testing.T) {
	for _, test := range idTests {
		got := ID(test.decoded[:])
		if got != test.encoded {
			t.Errorf("ID(%v) = %q want %q", test.decoded, got, test.encoded)
		}
	}
}

func TestDecodeHash256Ok(t *testing.T) {
	for _, test := range idTests {
		var got [32]byte
		err := DecodeHash256(test.encoded, &got)
		if err != nil {
			t.Errorf("DecodeHash256(%q) err = %v want nil", test.encoded, err)
		}
		if got != test.decoded {
			t.Errorf("DecodeHash256(%q) = %v want %v", test.encoded, got, test.decoded)
		}
	}
}

func TestDecodeHash256Pad(t *testing.T) {
	var got [32]byte
	err := DecodeHash256("1", &got)
	if err != nil {
		t.Errorf("DecodeHash256(%q) err = %v want nil", "1", err)
	}
	want := [32]byte{1}
	if got != want {
		t.Errorf("DecodeHash256(%q) = %v want %v", "1", got, want)
	}
}

func TestDecodeHash256Err(t *testing.T) {
	cases := []string{
		"xy", // non-hex chars
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // too long
	}

	for _, test := range cases {
		var x [32]byte
		err := DecodeHash256(test, &x)
		if err == nil {
			t.Errorf("DecodeHash256(%q) err = nil want non-nil error", test)
		}
	}
}

func TestEncodeIDTwice(t *testing.T) {
	h := []byte{1, 2, 3, 4}
	g0 := ID(h)
	g1 := ID(h)
	want := "04030201"
	if want != g0 {
		t.Errorf("ID(%v) (first) = %q want %q", h, g0, want)
	}
	if want != g1 {
		t.Errorf("ID(%v) (second) = %q want %q", h, g1, want)
	}
}
