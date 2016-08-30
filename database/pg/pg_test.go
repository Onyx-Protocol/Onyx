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
