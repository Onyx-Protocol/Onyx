package testutil

import "testing"

func TestDeepEqual(t *testing.T) {
	type s struct {
		a int
		b string
	}

	cases := []struct {
		a, b interface{}
		want bool
	}{
		{1, 1, true},
		{1, 2, false},
		{nil, nil, true},
		{nil, []byte{}, true},
		{nil, []byte{1}, false},
		{[]byte{1}, []byte{1}, true},
		{[]byte{1}, []byte{2}, false},
		{[]byte{1}, []byte{1, 2}, false},
		{[]byte{1}, []string{"1"}, false},
		{[3]byte{}, [4]byte{}, false},
		{[3]byte{1}, [3]byte{1, 0, 0}, true},
		{s{}, s{}, true},
		{s{a: 1}, s{}, false},
		{s{b: "foo"}, s{}, false},
		{"foo", "foo", true},
		{"foo", "bar", false},
		{"foo", nil, false},
	}

	for i, c := range cases {
		got := DeepEqual(c.a, c.b)
		if got != c.want {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}
