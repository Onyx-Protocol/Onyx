package ivy

import "testing"

func TestRequireAllParamsUsedInClauses(t *testing.T) {
	clauses := []*clause{
		&clause{
			statements: []statement{
				&verifyStatement{expr: varRef("foo")},
				&verifyStatement{
					expr: &binaryExpr{
						left:  varRef("foo"),
						right: varRef("bar"),
					},
				},
				&lockStatement{
					locked:  varRef("baz"),
					program: varRef("foo"),
				},
			},
		},
		&clause{
			statements: []statement{
				&verifyStatement{expr: varRef("foo")},
				&verifyStatement{
					expr: &binaryExpr{
						left:  varRef("foo"),
						right: varRef("plugh"),
					},
				},
				&lockStatement{
					locked:  varRef("xyzzy"),
					program: varRef("foo"),
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
			var params []*param
			for _, p := range c.params {
				params = append(params, &param{name: p})
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
