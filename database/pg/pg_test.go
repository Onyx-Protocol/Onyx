package pg

import (
	"net"
	"testing"
)

func TestResolveURI(t *testing.T) {
	addrs, err := net.LookupHost("example.com")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		input string
		want  string
	}{
		{"postgres:///foo", "postgres:///foo"},
		{"postgres://example.com/foo", "postgres://" + addrs[0] + "/foo"},
	}

	for _, c := range cases {
		res, err := resolveURI(c.input)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if res != c.want {
			t.Fatalf("resolveURI(%q) = %q, want %q", c.input, res, c.want)
		}

	}

}

func TestIsValidJSONB(t *testing.T) {
	cases := map[string]bool{
		`"hello"`: true,
		`{`:       false,
		`{"foo": ["bar", "baz"]}`:                    true,
		`{"bad": {"foo": "bar\u0000"}}`:              false,
		`{"bad": {"foo\u0000": "bar"}}`:              false,
		`{"bad": "\u0000"}`:                          false,
		`["hello", "world", "what is \u0000p?"]`:     false,
		`"` + string([]byte{0xff, 0xfe, 0xfd}) + `"`: false,
	}

	for b, want := range cases {
		t.Run(b, func(t *testing.T) {
			if got := IsValidJSONB([]byte(b)); got != want {
				t.Errorf("got %t want %t", got, want)
			}
		})
	}
}
