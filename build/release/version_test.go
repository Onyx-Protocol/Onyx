package release

import "testing"

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

func TestPrevver(t *testing.T) {
	cases := []struct{ v, p string }{
		{"1", ""},
		{"1.2", "1.1"},
		{"2", "1"},
		{"2.1", "2"},
		{"2.5", "2.4"},
		{"2.5.1", "2.5"},
		{"2.5.2", "2.5.1"},
		{"2rc1", "1"},
		{"2rc5", "1"},
	}

	for _, test := range cases {
		g := Prev(test.v)
		if g != test.p {
			t.Errorf("Prev(%q) = %q, want %q", test.v, g, test.p)
		}
	}
}
