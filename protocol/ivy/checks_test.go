package ivy

import "testing"

func TestRequireAllParamsUsedInClauses(t *testing.T) {
	clauses := []*clause{
		&clause{
			statements: []statement{
				&verifyStatement{expr: &varRef{name: "foo"}},
				&verifyStatement{
					expr: &binaryExpr{
						left:  &varRef{name: "foo"},
						right: &varRef{name: "bar"},
					},
				},
				&outputStatement{
					call: &call{
						fn: &varRef{name: "foo"},
						args: []expression{
							&varRef{name: "baz"},
						},
					},
				},
			},
		},
		&clause{
			statements: []statement{
				&verifyStatement{expr: &varRef{name: "foo"}},
				&verifyStatement{
					expr: &binaryExpr{
						left:  &varRef{name: "foo"},
						right: &varRef{name: "plugh"},
					},
				},
				&outputStatement{
					call: &call{
						fn: &varRef{name: "foo"},
						args: []expression{
							&varRef{name: "xyzzy"},
						},
					},
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

func TestProhibitNameCollisions(t *testing.T) {
	cases := []struct {
		name string
		inp  string
		want string
	}{
		{
			name: "keyword in contract name",
			inp:  "contract output() {}",
			want: "contract name \"output\" conflicts with keyword",
		},
		{
			name: "builtin in contract name",
			inp:  "contract sha3() {}",
			want: "contract name \"sha3\" conflicts with built-in function",
		},
		{
			name: "keyword in contract param",
			inp:  "contract foo(output: Integer) {}",
			want: "contract parameter \"output\" conflicts with keyword",
		},
		{
			name: "builtin in contract param",
			inp:  "contract foo(sha3: Integer) {}",
			want: "contract parameter \"sha3\" conflicts with built-in function",
		},
		{
			name: "contract name in contract param",
			inp:  "contract foo(foo: Integer) {}",
			want: "contract parameter \"foo\" conflicts with contract name",
		},
		{
			name: "keyword in clause name",
			inp:  "contract foo() { clause output() {} }",
			want: "clause name \"output\" conflicts with keyword",
		},
		{
			name: "builtin in clause name",
			inp:  "contract foo() { clause sha3() {} }",
			want: "clause name \"sha3\" conflicts with built-in function",
		},
		{
			name: "contract name in clause name",
			inp:  "contract foo() { clause foo() {} }",
			want: "clause name \"foo\" conflicts with contract name",
		},
		{
			name: "contract param in clause name",
			inp:  "contract foo(x: Integer) { clause x() {} }",
			want: "clause name \"x\" conflicts with contract parameter",
		},
		{
			name: "duplicate clause name",
			inp:  "contract foo() { clause bar() {} clause bar() {} }",
			want: "clause name \"bar\" conflicts with clause name",
		},
		{
			name: "keyword in clause param",
			inp:  "contract foo() { clause bar(output: Integer) {} }",
			want: "parameter \"output\" of clause \"bar\" conflicts with keyword",
		},
		{
			name: "builtin in clause param",
			inp:  "contract foo() { clause bar(sha3: Integer) {} }",
			want: "parameter \"sha3\" of clause \"bar\" conflicts with built-in function",
		},
		{
			name: "contract name in clause param",
			inp:  "contract foo() { clause bar(foo: Integer) {} }",
			want: "parameter \"foo\" of clause \"bar\" conflicts with contract name",
		},
		{
			name: "contract param in clause param",
			inp:  "contract foo(x: Integer) { clause bar(x: Integer) {} }",
			want: "parameter \"x\" of clause \"bar\" conflicts with contract parameter",
		},
		{
			name: "duplicate clause param",
			inp:  "contract foo() { clause bar(x: Integer, x: Integer) {} }",
			want: "parameter \"x\" of clause \"bar\" conflicts with clause parameter",
		},
		{
			name: "ok for different clauses to use the same parameter name",
			inp:  "contract foo() { clause bar(x: Integer) {} clause baz(x: Integer) {} }",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed, err := parse([]byte(c.inp))
			if err != nil {
				t.Fatal(err)
			}
			err = prohibitNameCollisions(parsed)
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

func TestRequireValueParam(t *testing.T) {
	cases := []struct {
		name string
		inp  string
		want string
	}{
		{
			name: "zero params",
			inp:  "contract foo() {}",
			want: "must have at least one contract parameter",
		},
		{
			name: "one non-Value param",
			inp:  "contract foo(x: Integer) {}",
			want: "final contract parameter has type \"Integer\" but should be Value",
		},
		{
			name: "two Value params",
			inp:  "contract foo(x: Value, y: Value) {}",
			want: "contract parameter 0 has type Value, but only the final parameter may",
		},
		{
			name: "ok",
			inp:  "contract foo(x: Integer, y: Value) {}",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed, err := parse([]byte(c.inp))
			if err != nil {
				t.Fatal(err)
			}
			err = requireValueParam(parsed)
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
