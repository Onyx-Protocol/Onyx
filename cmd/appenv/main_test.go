package main

import (
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	cases := []struct {
		list []string
		s    string
		w    bool
	}{
		{[]string{"a", "b", "c"}, "a", true},
		{[]string{"a", "b", "c"}, "c", true},
		{[]string{"a", "b", "c"}, "z", false},
		{nil, "a", false},
	}
	for _, test := range cases {
		g := contains(test.list, test.s)
		if g != test.w {
			t.Errorf("contains(%v, %v) = %v want %v", test.list, test.s, g, test.w)
		}
	}
}

func TestRemoveEnvNames(t *testing.T) {
	cases := [][3][]string{
		{{}, {}, {}},
		{{"a=1", "b=2"}, {"a"}, {"b=2"}},
		{{"a=1", "b=2"}, nil, {"a=1", "b=2"}},
		{{"a=1", "b=2"}, {"a="}, {"a=1", "b=2"}},
		{{"a=1", "b=2", "c=3"}, {"a"}, {"b=2", "c=3"}},
	}
	for _, test := range cases {
		env := make([]string, len(test[0]))
		copy(env, test[0]) // removeEnvNames modifies its argument
		names, w := test[1], test[2]
		g := removeEnvNames(env, names)
		if !reflect.DeepEqual(g, w) {
			t.Errorf("removeEnvNames(%v, %v) = %v want %v", test[0], names, g, w)
		}
	}
}
