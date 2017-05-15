package core

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
)

// Adapted from encoding/base32.
// See $GOROOT/src/encoding/base32/base32_test.go.
func TestB32encCrockford(t *testing.T) {
	cases := []struct {
		decoded, encoded string
	}{
		{"", ""},
		{"f", "CR"},
		{"fo", "CSQG"},
		{"foo", "CSQPY"},
		{"foob", "CSQPYRG"},
		{"fooba", "CSQPYRK1"},
		{"foobar", "CSQPYRK1E8"},
		{"leasure.", "DHJP2WVNE9JJW"},
		{"easure.", "CNGQ6XBJCMQ0"},
		{"asure.", "C5SQAWK55R"},
		{"sure.", "EDTQ4S9E"},
		{"sure", "EDTQ4S8"},
		{"sur", "EDTQ4"},
		{"su", "EDTG"},
		{
			"Twas brillig, and the slithy toves",
			"AHVP2WS0C9S6JV3CD5KJR831DSJ20X38CMG76V39EHM7J83MDXV6AWR",
		},
		{"\xDE\xAD\xBE\xEF", "VTPVXVR"},
		{"\x00\x11\x22\x33\x44\x55\x66\x77", "008J4CT4ANK7E"},
	}

	db := pgtest.NewTx(t)
	ctx := context.Background()
	for _, test := range cases {
		var got string
		err := db.QueryRowContext(ctx, `SELECT b32enc_crockford($1)`, test.decoded).Scan(&got)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if got != test.encoded {
			t.Errorf("b32enc_crockford(%q) = %q want %q", test.decoded, got, test.encoded)
		}
	}
}
