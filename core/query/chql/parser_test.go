package chql

import (
	"reflect"
	"testing"
)

func TestParseValid(t *testing.T) {
	testCases := []struct {
		q    string
		expr expr
	}{
		{
			q:    "'hello world'",
			expr: valueExpr{typ: tokString, value: "'hello world'"},
		},
		{
			q:    "2000",
			expr: valueExpr{typ: tokInteger, value: "2000"},
		},
		{
			q:    "0",
			expr: valueExpr{typ: tokInteger, value: "0"},
		},
		{
			q:    "0xff",
			expr: valueExpr{typ: tokInteger, value: "255"},
		},
		{
			q: "reference.recipient.email_address",
			expr: selectorExpr{
				ident: "email_address",
				objExpr: selectorExpr{
					ident:   "recipient",
					objExpr: attrExpr{attr: "reference"},
				},
			},
		},
		{
			q: "(reference.recipient).email_address",
			expr: selectorExpr{
				ident: "email_address",
				objExpr: parenExpr{
					inner: selectorExpr{
						ident:   "recipient",
						objExpr: attrExpr{attr: "reference"},
					},
				},
			},
		},
		{
			q: "2000 = 1000",
			expr: binaryExpr{
				op: binaryOps["="],
				l:  valueExpr{typ: tokInteger, value: "2000"},
				r:  valueExpr{typ: tokInteger, value: "1000"},
			},
		},
		{
			q: "INPUTS(asset_id = $1)",
			expr: envExpr{
				ident: "INPUTS",
				expr: binaryExpr{
					op: binaryOps["="],
					l:  attrExpr{attr: "asset_id"},
					r:  placeholderExpr{num: 1},
				},
			},
		},
		{
			q: "INPUTS(asset_id = $1) OR OUTPUTS(asset_id = 'abcdefg')",
			expr: binaryExpr{
				op: binaryOps["OR"],
				l: envExpr{
					ident: "INPUTS",
					expr: binaryExpr{
						op: binaryOps["="],
						l:  attrExpr{attr: "asset_id"},
						r:  placeholderExpr{num: 1},
					},
				},
				r: envExpr{
					ident: "OUTPUTS",
					expr: binaryExpr{
						op: binaryOps["="],
						l:  attrExpr{attr: "asset_id"},
						r:  valueExpr{typ: tokString, value: "'abcdefg'"},
					},
				},
			},
		},
		{
			q: "INPUTS(asset_tags.promissory_note AND account_tags.id = $1)",
			expr: envExpr{
				ident: "INPUTS",
				expr: binaryExpr{
					op: binaryOps["AND"],
					l: selectorExpr{
						objExpr: attrExpr{attr: "asset_tags"},
						ident:   "promissory_note",
					},
					r: binaryExpr{
						op: binaryOps["="],
						l: selectorExpr{
							objExpr: attrExpr{attr: "account_tags"},
							ident:   "id",
						},
						r: placeholderExpr{num: 1},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		expr, _, err := parse(tc.q)
		if err != nil {
			t.Errorf("%d: %s", i, err)
			continue
		}
		if !reflect.DeepEqual(expr, tc.expr) {
			t.Errorf("%d: parsing %q\ngot=\n%#v\nwant=\n%#v\n", i, tc.q, expr, tc.expr)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	testCases := []string{
		"123!",                                        // illegal !
		"INPUTS()",                                    // missing scope expr
		"INPUTS(account_tags.num = $a)",               // invalid placeholder num
		"0000124",                                     // no integer leading zeros
		`"double quotes"`,                             // double quotes not allowed
		"5 = $",                                       // $ without number
		"'unterminated string",                        // unterminated string
		`'strings do not allow \ backslash'`,          // illegal backslash
		"0x = 420",                                    // 0x without number
		"an_identifier another_identifier",            // two identifiers w/o an operator (trailing garbage)
		"inputs(account_tags.level = $1) or (1 == 1)", // lowercase 'or' (trailing garbage)
		"reference.(recipient.email_address)`",        // expected ident, got paren expr
	}
	for _, tc := range testCases {
		expr, _, err := parse(tc)
		if err == nil {
			t.Errorf("parse(%q) = %#v want error", tc, expr)
		}
	}
}
