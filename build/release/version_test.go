package release

import "testing"

func TestCheckVersion(t *testing.T) {
	cases := []struct {
		s  string
		ok bool
	}{
		{"", false},
		{"1", true},
		{"x", false},
		{"1.0", false},
		{"1.1", true},
		{"1.01", false},
		{"1.1rc0", false},
		{"1.1rc1", true},
		{"1.1rc1.1", false},
		{"1.1.1", true},
		{"1.1.1.1", false},
	}

	for _, test := range cases {
		gotOk := CheckVersion(test.s) == nil
		if gotOk != test.ok {
			t.Errorf("CheckVersion(%q) error is %v, want %v", test.s, gotOk, test.ok)
		}
	}
}

func TestLess(t *testing.T) {
	cases := [][]string{
		{"", "1"},
		{"1", "1.1"},
		{"1", "1.0.1"},
		{"1", "1.1.1"},
		{"1", "2"},
		{"1.1", "1.1.1"},
		{"1.1", "1.2"},
		{"1.1.1", "1.1.2"},
		{"1.1.1rc1", "1.1.1"},
		{"1.1.1rc1", "1.1.1rc2"},
		{"1.1.1rc2", "1.1.1"},
		{"1.1.2", "1.2.1"},
		{"1.1rc1", "1.1"},
		{"1.1rc1", "1.1rc2"},
		{"1.1rc2", "1.1"},
		{"1.2", "2.1"},
		{"1rc1", "1"},
		{"1rc1", "1rc2"},
		{"1rc2", "1"},
	}

	for _, test := range cases {
		ok := Less(test[0], test[1])
		if !ok {
			t.Errorf("Less(%q, %q) = false want true", test[0], test[1])
		}
	}
}

func TestPrevious(t *testing.T) {
	cases := []struct{ v, p string }{
		{"0.0.1", ""},
		{"0.0.2", "0.0.1"},
		{"0.1", ""},
		{"0.2", "0.1"},
		{"1", ""},
		{"1.0.1", "1"},
		{"1.2", "1.1"},
		{"2rc1", "1"},
		{"2rc5", "1"},
		{"2", "1"},
		{"2.1", "2"},
		{"2.5", "2.4"},
		{"2.5.1", "2.5"},
		{"2.5.2", "2.5.1"},
	}

	for _, test := range cases {
		g := Previous(test.v)
		if g != test.p {
			t.Errorf("Previous(%q) = %q, want %q", test.v, g, test.p)
			continue
		}
		if !Less(g, test.v) {
			t.Errorf("Previous(%q) = %q >= %q, want <", test.v, g, test.v)
		}
	}
}
