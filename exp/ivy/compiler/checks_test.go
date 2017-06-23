package compiler

import "testing"

func TestRequireAllParamsUsedInClauses(t *testing.T) {
	clauses := []*Clause{
		&Clause{
			Statements: []Statement{
				&VerifyStatement{Expr: VarRef("foo")},
				&VerifyStatement{
					Expr: &BinaryExpr{
						left:  VarRef("foo"),
						right: VarRef("bar"),
					},
				},
				&LockStatement{
					Locked:  VarRef("baz"),
					Program: VarRef("foo"),
				},
			},
		},
		&Clause{
			Statements: []Statement{
				&VerifyStatement{Expr: VarRef("foo")},
				&VerifyStatement{
					Expr: &BinaryExpr{
						left:  VarRef("foo"),
						right: VarRef("plugh"),
					},
				},
				&LockStatement{
					Locked:  VarRef("xyzzy"),
					Program: VarRef("foo"),
				},
			},
		},
	}

	cases := []struct {
		name   string
		params []string
		want   string
	}{
		{
			name:   "contract param used in both clauses",
			params: []string{"foo"},
		},
		{
			name:   "contract param used in one clause",
			params: []string{"bar"},
		},
		{
			name:   "contract param used in no clauses",
			params: []string{"y2"},
			want:   "parameter \"y2\" is unused",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var params []*Param
			for _, p := range c.params {
				params = append(params, &Param{Name: p})
			}
			err := requireAllParamsUsedInClauses(params, clauses)
			if err == nil && c.want == "" {
				return
			}
			if err == nil {
				t.Errorf("got err==nil, want %s", c.want)
				return
			}
			if err.Error() != c.want {
				t.Errorf("got %s, want %s", err, c.want)
			}
		})
	}
}
