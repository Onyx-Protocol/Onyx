package main

import "testing"

func TestParseExpr(t *testing.T) {
	cases := []struct{ expr, want string }{
		{`-3+4`, `binaryExpr{lhs: -3, rhs: 4, op: +}`},
		{`!3+4`, `binaryExpr{lhs: unaryExpr{expr: 3, op: !}, rhs: 4, op: +}`},
	}

	for _, test := range cases {
		p := &parser{buf: []byte(test.expr)}
		got := parseExpr(p).String()
		if got != test.want {
			t.Errorf("parseExpr(%q) = %s want %s", test.expr, got, test.want)
		}
	}
}
