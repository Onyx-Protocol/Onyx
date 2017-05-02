package name

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAttributeValue(t *testing.T) {
	cases := []struct {
		encoded string
		decoded interface{}
	}{
		{"x", "x"},
		{`jsmith`, `jsmith`},
		{`example`, `example`},
		{`net`, `net`},
		{`J. Smith`, `J. Smith`},
		{`Sales`, `Sales`},
		{`James \"Jim\" Smith\, III`, `James "Jim" Smith, III`},
		{"Before\nAfter", "Before\nAfter"},
		{`#04024869`, []byte{0x48, 0x69}},
		{`#04024869`, []byte{0x48, 0x69}},
		{`Test`, `Test`},
		{`GB`, `GB`},
		{`Lučić`, `Lučić`},
		{`West`, `West`},
		{`Engineering`, `Engineering`},
		{`Acme Corp.`, `Acme Corp.`},
		{`localhost`, `localhost`},
		{`Chain\, Inc.`, `Chain, Inc.`},
	}

	for _, test := range cases {
		t.Run(test.encoded, func(t *testing.T) {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatal(rec)
				}
			}()

			r := strings.NewReader(test.encoded)
			got := parseAttributeValue(r)
			if !reflect.DeepEqual(got, test.decoded) {
				t.Errorf("parseAttributeValue(%q) = %+v want %+v", test.encoded, got, test.decoded)
			}
		})
	}
}

func TestUnescape(t *testing.T) {
	cases := []struct {
		esc   string
		unesc string
	}{
		{"x", "x"},
		{`jsmith`, `jsmith`},
		{`example`, `example`},
		{`net`, `net`},
		{`J. Smith`, `J. Smith`},
		{`Sales`, `Sales`},
		{`James \"Jim\" Smith\, III`, `James "Jim" Smith, III`},
		{"Before\nAfter", "Before\nAfter"},
		{`Test`, `Test`},
		{`GB`, `GB`},
		{`Lučić`, `Lučić`},
		{`West`, `West`},
		{`Engineering`, `Engineering`},
		{`Acme Corp.`, `Acme Corp.`},
		{`localhost`, `localhost`},
		{`Chain\, Inc.`, `Chain, Inc.`},
		{`\01`, "\x01"},
		{`\61`, "\x61"},
		{`\ab`, "\xab"},
	}

	for _, test := range cases {
		t.Run(test.esc, func(t *testing.T) {
			got := unescaper.Replace(test.esc)
			if !reflect.DeepEqual(got, test.unesc) {
				t.Errorf("unescape(%#q) = %#q want %#q", test.esc, got, test.unesc)
			}
		})
	}
}
