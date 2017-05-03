package ivy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

var g = &grammar{
	rules: []*rule{
		{
			name: "Contract",
			pos:  position{line: 5, col: 1, offset: 17},
			expr: &actionExpr{
				pos: position{line: 5, col: 12, offset: 28},
				run: (*parser).callonContract1,
				expr: &seqExpr{
					pos: position{line: 5, col: 13, offset: 29},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 5, col: 13, offset: 29},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 15, offset: 31},
							val:        "contract",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 26, offset: 42},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 29, offset: 45},
							label: "name",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 34, offset: 50},
								name: "Identifier",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 45, offset: 61},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 47, offset: 63},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 51, offset: 67},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 53, offset: 69},
							label: "params",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 60, offset: 76},
								name: "Params",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 67, offset: 83},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 69, offset: 85},
							val:        ")",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 73, offset: 89},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 75, offset: 91},
							val:        "{",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 79, offset: 95},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 81, offset: 97},
							label: "clauses",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 89, offset: 105},
								name: "Clauses",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 97, offset: 113},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 99, offset: 115},
							val:        "}",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Params",
			pos:  position{line: 9, col: 1, offset: 168},
			expr: &choiceExpr{
				pos: position{line: 9, col: 10, offset: 177},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 9, col: 10, offset: 177},
						run: (*parser).callonParams2,
						expr: &seqExpr{
							pos: position{line: 9, col: 11, offset: 178},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 9, col: 11, offset: 178},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 9, col: 17, offset: 184},
										name: "Params1Type",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 9, col: 29, offset: 196},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 9, col: 31, offset: 198},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 9, col: 35, offset: 202},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 9, col: 37, offset: 204},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 9, col: 42, offset: 209},
										name: "Params",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 11, col: 5, offset: 259},
						run: (*parser).callonParams11,
						expr: &labeledExpr{
							pos:   position{line: 11, col: 5, offset: 259},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 11, col: 10, offset: 264},
								name: "Params1Type",
							},
						},
					},
					&actionExpr{
						pos: position{line: 13, col: 5, offset: 301},
						run: (*parser).callonParams14,
						expr: &ruleRefExpr{
							pos:  position{line: 13, col: 5, offset: 301},
							name: "Nothing",
						},
					},
				},
			},
		},
		{
			name: "Params1Type",
			pos:  position{line: 17, col: 1, offset: 339},
			expr: &choiceExpr{
				pos: position{line: 17, col: 15, offset: 353},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 17, col: 15, offset: 353},
						run: (*parser).callonParams1Type2,
						expr: &seqExpr{
							pos: position{line: 17, col: 16, offset: 354},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 17, col: 16, offset: 354},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 17, col: 22, offset: 360},
										name: "Identifier",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 17, col: 33, offset: 371},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 17, col: 35, offset: 373},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 17, col: 39, offset: 377},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 17, col: 41, offset: 379},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 17, col: 46, offset: 384},
										name: "Params1Type",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 19, col: 5, offset: 434},
						run: (*parser).callonParams1Type11,
						expr: &seqExpr{
							pos: position{line: 19, col: 6, offset: 435},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 19, col: 6, offset: 435},
									label: "only",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 11, offset: 440},
										name: "Identifier",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 19, col: 22, offset: 451},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 19, col: 24, offset: 453},
									val:        ":",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 19, col: 28, offset: 457},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 19, col: 30, offset: 459},
									label: "typ",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 34, offset: 463},
										name: "Type",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Type",
			pos:  position{line: 23, col: 1, offset: 502},
			expr: &actionExpr{
				pos: position{line: 23, col: 8, offset: 509},
				run: (*parser).callonType1,
				expr: &choiceExpr{
					pos: position{line: 23, col: 9, offset: 510},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 23, col: 9, offset: 510},
							val:        "String",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 20, offset: 521},
							val:        "Integer",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 32, offset: 533},
							val:        "Hash",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 41, offset: 542},
							val:        "AssetAmount",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 57, offset: 558},
							val:        "Program",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 69, offset: 570},
							val:        "Value",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Clauses",
			pos:  position{line: 27, col: 1, offset: 613},
			expr: &choiceExpr{
				pos: position{line: 27, col: 11, offset: 623},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 27, col: 11, offset: 623},
						run: (*parser).callonClauses2,
						expr: &seqExpr{
							pos: position{line: 27, col: 12, offset: 624},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 27, col: 12, offset: 624},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 27, col: 18, offset: 630},
										name: "Clause",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 27, col: 25, offset: 637},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 27, col: 27, offset: 639},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 27, col: 32, offset: 644},
										name: "Clauses",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 29, col: 5, offset: 695},
						run: (*parser).callonClauses9,
						expr: &labeledExpr{
							pos:   position{line: 29, col: 5, offset: 695},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 29, col: 10, offset: 700},
								name: "Clause",
							},
						},
					},
				},
			},
		},
		{
			name: "Clause",
			pos:  position{line: 33, col: 1, offset: 737},
			expr: &actionExpr{
				pos: position{line: 33, col: 10, offset: 746},
				run: (*parser).callonClause1,
				expr: &seqExpr{
					pos: position{line: 33, col: 11, offset: 747},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 33, col: 11, offset: 747},
							val:        "clause",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 20, offset: 756},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 23, offset: 759},
							label: "name",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 28, offset: 764},
								name: "Identifier",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 39, offset: 775},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 41, offset: 777},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 45, offset: 781},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 47, offset: 783},
							label: "params",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 54, offset: 790},
								name: "Params",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 61, offset: 797},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 63, offset: 799},
							val:        ")",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 67, offset: 803},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 69, offset: 805},
							val:        "{",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 73, offset: 809},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 75, offset: 811},
							label: "statements",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 86, offset: 822},
								name: "Statements",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 97, offset: 833},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 99, offset: 835},
							val:        "}",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Statements",
			pos:  position{line: 37, col: 1, offset: 889},
			expr: &choiceExpr{
				pos: position{line: 37, col: 14, offset: 902},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 37, col: 14, offset: 902},
						run: (*parser).callonStatements2,
						expr: &seqExpr{
							pos: position{line: 37, col: 15, offset: 903},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 37, col: 15, offset: 903},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 37, col: 21, offset: 909},
										name: "Statement",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 37, col: 31, offset: 919},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 37, col: 33, offset: 921},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 37, col: 38, offset: 926},
										name: "Statements",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 39, col: 5, offset: 983},
						run: (*parser).callonStatements9,
						expr: &labeledExpr{
							pos:   position{line: 39, col: 5, offset: 983},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 10, offset: 988},
								name: "Statement",
							},
						},
					},
				},
			},
		},
		{
			name: "Statement",
			pos:  position{line: 43, col: 1, offset: 1031},
			expr: &actionExpr{
				pos: position{line: 43, col: 13, offset: 1043},
				run: (*parser).callonStatement1,
				expr: &labeledExpr{
					pos:   position{line: 43, col: 13, offset: 1043},
					label: "s",
					expr: &choiceExpr{
						pos: position{line: 43, col: 16, offset: 1046},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 43, col: 16, offset: 1046},
								name: "VerifyStatement",
							},
							&ruleRefExpr{
								pos:  position{line: 43, col: 34, offset: 1064},
								name: "OutputStatement",
							},
							&ruleRefExpr{
								pos:  position{line: 43, col: 52, offset: 1082},
								name: "ReturnStatement",
							},
						},
					},
				},
			},
		},
		{
			name: "VerifyStatement",
			pos:  position{line: 47, col: 1, offset: 1120},
			expr: &actionExpr{
				pos: position{line: 47, col: 19, offset: 1138},
				run: (*parser).callonVerifyStatement1,
				expr: &seqExpr{
					pos: position{line: 47, col: 20, offset: 1139},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 47, col: 20, offset: 1139},
							val:        "verify",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 47, col: 29, offset: 1148},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 47, col: 32, offset: 1151},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 47, col: 37, offset: 1156},
								name: "Expr",
							},
						},
					},
				},
			},
		},
		{
			name: "OutputStatement",
			pos:  position{line: 51, col: 1, offset: 1191},
			expr: &actionExpr{
				pos: position{line: 51, col: 19, offset: 1209},
				run: (*parser).callonOutputStatement1,
				expr: &seqExpr{
					pos: position{line: 51, col: 20, offset: 1210},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 51, col: 20, offset: 1210},
							val:        "output",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 29, offset: 1219},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 51, col: 32, offset: 1222},
							label: "callExpr",
							expr: &ruleRefExpr{
								pos:  position{line: 51, col: 41, offset: 1231},
								name: "Call",
							},
						},
					},
				},
			},
		},
		{
			name: "ReturnStatement",
			pos:  position{line: 55, col: 1, offset: 1270},
			expr: &actionExpr{
				pos: position{line: 55, col: 19, offset: 1288},
				run: (*parser).callonReturnStatement1,
				expr: &seqExpr{
					pos: position{line: 55, col: 20, offset: 1289},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 55, col: 20, offset: 1289},
							val:        "return",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 55, col: 29, offset: 1298},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 55, col: 32, offset: 1301},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 55, col: 37, offset: 1306},
								name: "Expr",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr",
			pos:  position{line: 59, col: 1, offset: 1341},
			expr: &actionExpr{
				pos: position{line: 59, col: 8, offset: 1348},
				run: (*parser).callonExpr1,
				expr: &labeledExpr{
					pos:   position{line: 59, col: 8, offset: 1348},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 59, col: 11, offset: 1351},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 59, col: 11, offset: 1351},
								name: "ComparisonExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 59, col: 28, offset: 1368},
								name: "Expr2",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr2",
			pos:  position{line: 63, col: 1, offset: 1396},
			expr: &actionExpr{
				pos: position{line: 63, col: 9, offset: 1404},
				run: (*parser).callonExpr21,
				expr: &labeledExpr{
					pos:   position{line: 63, col: 9, offset: 1404},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 63, col: 12, offset: 1407},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 63, col: 12, offset: 1407},
								name: "BinaryExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 63, col: 25, offset: 1420},
								name: "Expr3",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr3",
			pos:  position{line: 67, col: 1, offset: 1448},
			expr: &actionExpr{
				pos: position{line: 67, col: 9, offset: 1456},
				run: (*parser).callonExpr31,
				expr: &labeledExpr{
					pos:   position{line: 67, col: 9, offset: 1456},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 67, col: 12, offset: 1459},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 67, col: 12, offset: 1459},
								name: "UnaryExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 67, col: 24, offset: 1471},
								name: "Expr4",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr4",
			pos:  position{line: 71, col: 1, offset: 1499},
			expr: &choiceExpr{
				pos: position{line: 71, col: 9, offset: 1507},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 71, col: 9, offset: 1507},
						run: (*parser).callonExpr42,
						expr: &labeledExpr{
							pos:   position{line: 71, col: 9, offset: 1507},
							label: "e",
							expr: &choiceExpr{
								pos: position{line: 71, col: 12, offset: 1510},
								alternatives: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 71, col: 12, offset: 1510},
										name: "Call",
									},
									&ruleRefExpr{
										pos:  position{line: 71, col: 19, offset: 1517},
										name: "Literal",
									},
									&ruleRefExpr{
										pos:  position{line: 71, col: 29, offset: 1527},
										name: "Expr5",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 73, col: 5, offset: 1556},
						run: (*parser).callonExpr48,
						expr: &seqExpr{
							pos: position{line: 73, col: 6, offset: 1557},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 73, col: 6, offset: 1557},
									val:        "(",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 73, col: 10, offset: 1561},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 73, col: 12, offset: 1563},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 73, col: 14, offset: 1565},
										name: "Expr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 73, col: 19, offset: 1570},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 73, col: 21, offset: 1572},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Expr5",
			pos:  position{line: 77, col: 1, offset: 1598},
			expr: &actionExpr{
				pos: position{line: 77, col: 9, offset: 1606},
				run: (*parser).callonExpr51,
				expr: &labeledExpr{
					pos:   position{line: 77, col: 9, offset: 1606},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 77, col: 12, offset: 1609},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 77, col: 12, offset: 1609},
								name: "PropRef",
							},
							&ruleRefExpr{
								pos:  position{line: 77, col: 22, offset: 1619},
								name: "VarRef",
							},
						},
					},
				},
			},
		},
		{
			name: "PropRef",
			pos:  position{line: 81, col: 1, offset: 1648},
			expr: &actionExpr{
				pos: position{line: 81, col: 11, offset: 1658},
				run: (*parser).callonPropRef1,
				expr: &seqExpr{
					pos: position{line: 81, col: 12, offset: 1659},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 81, col: 12, offset: 1659},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 81, col: 14, offset: 1661},
								name: "VarRef",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 81, col: 21, offset: 1668},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 81, col: 23, offset: 1670},
							val:        ".",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 81, col: 27, offset: 1674},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 81, col: 29, offset: 1676},
							label: "property",
							expr: &ruleRefExpr{
								pos:  position{line: 81, col: 38, offset: 1685},
								name: "Identifier",
							},
						},
					},
				},
			},
		},
		{
			name: "VarRef",
			pos:  position{line: 85, col: 1, offset: 1734},
			expr: &actionExpr{
				pos: position{line: 85, col: 10, offset: 1743},
				run: (*parser).callonVarRef1,
				expr: &labeledExpr{
					pos:   position{line: 85, col: 10, offset: 1743},
					label: "name",
					expr: &ruleRefExpr{
						pos:  position{line: 85, col: 15, offset: 1748},
						name: "Identifier",
					},
				},
			},
		},
		{
			name: "ComparisonExpr",
			pos:  position{line: 90, col: 1, offset: 1807},
			expr: &actionExpr{
				pos: position{line: 90, col: 18, offset: 1824},
				run: (*parser).callonComparisonExpr1,
				expr: &seqExpr{
					pos: position{line: 90, col: 19, offset: 1825},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 90, col: 19, offset: 1825},
							label: "left",
							expr: &ruleRefExpr{
								pos:  position{line: 90, col: 24, offset: 1830},
								name: "Expr2",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 90, col: 30, offset: 1836},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 90, col: 32, offset: 1838},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 90, col: 35, offset: 1841},
								name: "ComparisonOperator",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 90, col: 54, offset: 1860},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 90, col: 56, offset: 1862},
							label: "right",
							expr: &ruleRefExpr{
								pos:  position{line: 90, col: 62, offset: 1868},
								name: "Expr2",
							},
						},
					},
				},
			},
		},
		{
			name: "ComparisonOperator",
			pos:  position{line: 94, col: 1, offset: 1919},
			expr: &actionExpr{
				pos: position{line: 94, col: 22, offset: 1940},
				run: (*parser).callonComparisonOperator1,
				expr: &choiceExpr{
					pos: position{line: 94, col: 23, offset: 1941},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 94, col: 23, offset: 1941},
							val:        "==",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 94, col: 30, offset: 1948},
							val:        "!=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 94, col: 37, offset: 1955},
							val:        "<=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 94, col: 44, offset: 1962},
							val:        ">=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 94, col: 51, offset: 1969},
							val:        "<",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 94, col: 57, offset: 1975},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "BinaryExpr",
			pos:  position{line: 99, col: 1, offset: 2034},
			expr: &actionExpr{
				pos: position{line: 99, col: 14, offset: 2047},
				run: (*parser).callonBinaryExpr1,
				expr: &seqExpr{
					pos: position{line: 99, col: 15, offset: 2048},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 99, col: 15, offset: 2048},
							label: "partials",
							expr: &ruleRefExpr{
								pos:  position{line: 99, col: 24, offset: 2057},
								name: "PartialBinaryExprs",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 99, col: 43, offset: 2076},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 99, col: 45, offset: 2078},
							label: "right",
							expr: &ruleRefExpr{
								pos:  position{line: 99, col: 51, offset: 2084},
								name: "Expr3",
							},
						},
					},
				},
			},
		},
		{
			name: "PartialBinaryExprs",
			pos:  position{line: 103, col: 1, offset: 2158},
			expr: &choiceExpr{
				pos: position{line: 103, col: 22, offset: 2179},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 103, col: 22, offset: 2179},
						run: (*parser).callonPartialBinaryExprs2,
						expr: &seqExpr{
							pos: position{line: 103, col: 23, offset: 2180},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 103, col: 23, offset: 2180},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 103, col: 29, offset: 2186},
										name: "PartialBinaryExpr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 103, col: 47, offset: 2204},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 103, col: 49, offset: 2206},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 103, col: 54, offset: 2211},
										name: "PartialBinaryExprs",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 105, col: 5, offset: 2284},
						run: (*parser).callonPartialBinaryExprs9,
						expr: &labeledExpr{
							pos:   position{line: 105, col: 5, offset: 2284},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 105, col: 10, offset: 2289},
								name: "PartialBinaryExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "PartialBinaryExpr",
			pos:  position{line: 109, col: 1, offset: 2348},
			expr: &actionExpr{
				pos: position{line: 109, col: 21, offset: 2368},
				run: (*parser).callonPartialBinaryExpr1,
				expr: &seqExpr{
					pos: position{line: 109, col: 22, offset: 2369},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 109, col: 22, offset: 2369},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 109, col: 27, offset: 2374},
								name: "Expr3",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 109, col: 33, offset: 2380},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 109, col: 35, offset: 2382},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 109, col: 38, offset: 2385},
								name: "BinaryOperator",
							},
						},
					},
				},
			},
		},
		{
			name: "BinaryOperator",
			pos:  position{line: 113, col: 1, offset: 2445},
			expr: &actionExpr{
				pos: position{line: 113, col: 18, offset: 2462},
				run: (*parser).callonBinaryOperator1,
				expr: &choiceExpr{
					pos: position{line: 113, col: 19, offset: 2463},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 113, col: 19, offset: 2463},
							val:        "+",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 113, col: 25, offset: 2469},
							val:        "-",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "UnaryExpr",
			pos:  position{line: 117, col: 1, offset: 2508},
			expr: &actionExpr{
				pos: position{line: 117, col: 13, offset: 2520},
				run: (*parser).callonUnaryExpr1,
				expr: &seqExpr{
					pos: position{line: 117, col: 14, offset: 2521},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 117, col: 14, offset: 2521},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 117, col: 17, offset: 2524},
								name: "UnaryOperator",
							},
						},
						&labeledExpr{
							pos:   position{line: 117, col: 31, offset: 2538},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 117, col: 36, offset: 2543},
								name: "Expr4",
							},
						},
					},
				},
			},
		},
		{
			name: "UnaryOperator",
			pos:  position{line: 121, col: 1, offset: 2586},
			expr: &actionExpr{
				pos: position{line: 121, col: 17, offset: 2602},
				run: (*parser).callonUnaryOperator1,
				expr: &choiceExpr{
					pos: position{line: 121, col: 18, offset: 2603},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 121, col: 18, offset: 2603},
							val:        "-",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 121, col: 24, offset: 2609},
							val:        "!",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Call",
			pos:  position{line: 125, col: 1, offset: 2648},
			expr: &actionExpr{
				pos: position{line: 125, col: 8, offset: 2655},
				run: (*parser).callonCall1,
				expr: &seqExpr{
					pos: position{line: 125, col: 9, offset: 2656},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 125, col: 9, offset: 2656},
							label: "fn",
							expr: &ruleRefExpr{
								pos:  position{line: 125, col: 12, offset: 2659},
								name: "Expr5",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 125, col: 18, offset: 2665},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 125, col: 20, offset: 2667},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 125, col: 24, offset: 2671},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 125, col: 26, offset: 2673},
							label: "args",
							expr: &ruleRefExpr{
								pos:  position{line: 125, col: 31, offset: 2678},
								name: "Args",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 125, col: 36, offset: 2683},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 125, col: 38, offset: 2685},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Args",
			pos:  position{line: 129, col: 1, offset: 2721},
			expr: &choiceExpr{
				pos: position{line: 129, col: 8, offset: 2728},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 129, col: 8, offset: 2728},
						run: (*parser).callonArgs2,
						expr: &seqExpr{
							pos: position{line: 129, col: 9, offset: 2729},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 129, col: 9, offset: 2729},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 129, col: 15, offset: 2735},
										name: "Expr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 129, col: 20, offset: 2740},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 129, col: 22, offset: 2742},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 129, col: 26, offset: 2746},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 129, col: 28, offset: 2748},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 129, col: 33, offset: 2753},
										name: "Args",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 131, col: 5, offset: 2798},
						run: (*parser).callonArgs11,
						expr: &labeledExpr{
							pos:   position{line: 131, col: 5, offset: 2798},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 131, col: 10, offset: 2803},
								name: "Expr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 133, col: 5, offset: 2836},
						run: (*parser).callonArgs14,
						expr: &ruleRefExpr{
							pos:  position{line: 133, col: 5, offset: 2836},
							name: "Nothing",
						},
					},
				},
			},
		},
		{
			name: "Literal",
			pos:  position{line: 137, col: 1, offset: 2878},
			expr: &actionExpr{
				pos: position{line: 137, col: 11, offset: 2888},
				run: (*parser).callonLiteral1,
				expr: &labeledExpr{
					pos:   position{line: 137, col: 11, offset: 2888},
					label: "val",
					expr: &choiceExpr{
						pos: position{line: 137, col: 16, offset: 2893},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 137, col: 16, offset: 2893},
								name: "Integer",
							},
							&ruleRefExpr{
								pos:  position{line: 137, col: 26, offset: 2903},
								name: "Boolean",
							},
						},
					},
				},
			},
		},
		{
			name: "Integer",
			pos:  position{line: 141, col: 1, offset: 2935},
			expr: &actionExpr{
				pos: position{line: 141, col: 11, offset: 2945},
				run: (*parser).callonInteger1,
				expr: &seqExpr{
					pos: position{line: 141, col: 12, offset: 2946},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 141, col: 12, offset: 2946},
							expr: &litMatcher{
								pos:        position{line: 141, col: 12, offset: 2946},
								val:        "-",
								ignoreCase: false,
							},
						},
						&oneOrMoreExpr{
							pos: position{line: 141, col: 17, offset: 2951},
							expr: &charClassMatcher{
								pos:        position{line: 141, col: 17, offset: 2951},
								val:        "[0-9]",
								ranges:     []rune{'0', '9'},
								ignoreCase: false,
								inverted:   false,
							},
						},
					},
				},
			},
		},
		{
			name: "Boolean",
			pos:  position{line: 145, col: 1, offset: 2991},
			expr: &actionExpr{
				pos: position{line: 145, col: 11, offset: 3001},
				run: (*parser).callonBoolean1,
				expr: &choiceExpr{
					pos: position{line: 145, col: 12, offset: 3002},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 145, col: 12, offset: 3002},
							val:        "true",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 145, col: 21, offset: 3011},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Identifier",
			pos:  position{line: 149, col: 1, offset: 3052},
			expr: &actionExpr{
				pos: position{line: 149, col: 14, offset: 3065},
				run: (*parser).callonIdentifier1,
				expr: &seqExpr{
					pos: position{line: 149, col: 14, offset: 3065},
					exprs: []interface{}{
						&charClassMatcher{
							pos:        position{line: 149, col: 14, offset: 3065},
							val:        "[A-Za-z]",
							ranges:     []rune{'A', 'Z', 'a', 'z'},
							ignoreCase: false,
							inverted:   false,
						},
						&zeroOrMoreExpr{
							pos: position{line: 149, col: 22, offset: 3073},
							expr: &charClassMatcher{
								pos:        position{line: 149, col: 22, offset: 3073},
								val:        "[A-Za-z0-9]",
								ranges:     []rune{'A', 'Z', 'a', 'z', '0', '9'},
								ignoreCase: false,
								inverted:   false,
							},
						},
					},
				},
			},
		},
		{
			name: "Whitespace",
			pos:  position{line: 153, col: 1, offset: 3120},
			expr: &oneOrMoreExpr{
				pos: position{line: 153, col: 14, offset: 3133},
				expr: &charClassMatcher{
					pos:        position{line: 153, col: 14, offset: 3133},
					val:        "[ \\t\\n\\r]",
					chars:      []rune{' ', '\t', '\n', '\r'},
					ignoreCase: false,
					inverted:   false,
				},
			},
		},
		{
			name: "Comment",
			pos:  position{line: 155, col: 1, offset: 3145},
			expr: &seqExpr{
				pos: position{line: 155, col: 11, offset: 3155},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 155, col: 11, offset: 3155},
						val:        "#",
						ignoreCase: false,
					},
					&zeroOrMoreExpr{
						pos: position{line: 155, col: 15, offset: 3159},
						expr: &charClassMatcher{
							pos:        position{line: 155, col: 15, offset: 3159},
							val:        "[^\\n\\r]",
							chars:      []rune{'\n', '\r'},
							ignoreCase: false,
							inverted:   true,
						},
					},
				},
			},
		},
		{
			name: "_",
			pos:  position{line: 157, col: 1, offset: 3169},
			expr: &zeroOrMoreExpr{
				pos: position{line: 157, col: 5, offset: 3173},
				expr: &choiceExpr{
					pos: position{line: 157, col: 6, offset: 3174},
					alternatives: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 157, col: 6, offset: 3174},
							name: "Whitespace",
						},
						&ruleRefExpr{
							pos:  position{line: 157, col: 19, offset: 3187},
							name: "Comment",
						},
					},
				},
			},
		},
		{
			name: "__",
			pos:  position{line: 159, col: 1, offset: 3198},
			expr: &oneOrMoreExpr{
				pos: position{line: 159, col: 6, offset: 3203},
				expr: &choiceExpr{
					pos: position{line: 159, col: 7, offset: 3204},
					alternatives: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 159, col: 7, offset: 3204},
							name: "Whitespace",
						},
						&ruleRefExpr{
							pos:  position{line: 159, col: 20, offset: 3217},
							name: "Comment",
						},
					},
				},
			},
		},
		{
			name: "Nothing",
			pos:  position{line: 161, col: 1, offset: 3228},
			expr: &ruleRefExpr{
				pos:  position{line: 161, col: 11, offset: 3238},
				name: "_",
			},
		},
	},
}

func (c *current) onContract1(name, params, clauses interface{}) (interface{}, error) {
	return mkContract(name, params, clauses)
}

func (p *parser) callonContract1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onContract1(stack["name"], stack["params"], stack["clauses"])
}

func (c *current) onParams2(first, rest interface{}) (interface{}, error) {
	return prependParams(first, rest)
}

func (p *parser) callonParams2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onParams2(stack["first"], stack["rest"])
}

func (c *current) onParams11(only interface{}) (interface{}, error) {
	return only, nil
}

func (p *parser) callonParams11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onParams11(stack["only"])
}

func (c *current) onParams14() (interface{}, error) {
	return []*param{}, nil
}

func (p *parser) callonParams14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onParams14()
}

func (c *current) onParams1Type2(first, rest interface{}) (interface{}, error) {
	return mkParams(first, rest)
}

func (p *parser) callonParams1Type2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onParams1Type2(stack["first"], stack["rest"])
}

func (c *current) onParams1Type11(only, typ interface{}) (interface{}, error) {
	return mkParam(only, typ)
}

func (p *parser) callonParams1Type11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onParams1Type11(stack["only"], stack["typ"])
}

func (c *current) onType1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonType1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onType1()
}

func (c *current) onClauses2(first, rest interface{}) (interface{}, error) {
	return prependClause(first, rest)
}

func (p *parser) callonClauses2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onClauses2(stack["first"], stack["rest"])
}

func (c *current) onClauses9(only interface{}) (interface{}, error) {
	return mkClauses(only)
}

func (p *parser) callonClauses9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onClauses9(stack["only"])
}

func (c *current) onClause1(name, params, statements interface{}) (interface{}, error) {
	return mkClause(name, params, statements)
}

func (p *parser) callonClause1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onClause1(stack["name"], stack["params"], stack["statements"])
}

func (c *current) onStatements2(first, rest interface{}) (interface{}, error) {
	return prependStatement(first, rest)
}

func (p *parser) callonStatements2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onStatements2(stack["first"], stack["rest"])
}

func (c *current) onStatements9(only interface{}) (interface{}, error) {
	return mkStatements(only)
}

func (p *parser) callonStatements9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onStatements9(stack["only"])
}

func (c *current) onStatement1(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callonStatement1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onStatement1(stack["s"])
}

func (c *current) onVerifyStatement1(expr interface{}) (interface{}, error) {
	return mkVerify(expr)
}

func (p *parser) callonVerifyStatement1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onVerifyStatement1(stack["expr"])
}

func (c *current) onOutputStatement1(callExpr interface{}) (interface{}, error) {
	return mkOutput(callExpr)
}

func (p *parser) callonOutputStatement1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onOutputStatement1(stack["callExpr"])
}

func (c *current) onReturnStatement1(expr interface{}) (interface{}, error) {
	return mkReturn(expr)
}

func (p *parser) callonReturnStatement1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onReturnStatement1(stack["expr"])
}

func (c *current) onExpr1(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr1(stack["e"])
}

func (c *current) onExpr21(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr21() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr21(stack["e"])
}

func (c *current) onExpr31(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr31() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr31(stack["e"])
}

func (c *current) onExpr42(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr42() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr42(stack["e"])
}

func (c *current) onExpr48(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr48() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr48(stack["e"])
}

func (c *current) onExpr51(e interface{}) (interface{}, error) {
	return e, nil
}

func (p *parser) callonExpr51() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onExpr51(stack["e"])
}

func (c *current) onPropRef1(v, property interface{}) (interface{}, error) {
	return mkPropRef(v, property)
}

func (p *parser) callonPropRef1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onPropRef1(stack["v"], stack["property"])
}

func (c *current) onVarRef1(name interface{}) (interface{}, error) {
	return mkVarRef(name)
}

func (p *parser) callonVarRef1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onVarRef1(stack["name"])
}

func (c *current) onComparisonExpr1(left, op, right interface{}) (interface{}, error) {
	return mkBinaryExpr(left, op, right)
}

func (p *parser) callonComparisonExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onComparisonExpr1(stack["left"], stack["op"], stack["right"])
}

func (c *current) onComparisonOperator1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonComparisonOperator1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onComparisonOperator1()
}

func (c *current) onBinaryExpr1(partials, right interface{}) (interface{}, error) {
	return binaryExprFromPartials(partials, right.(expression))
}

func (p *parser) callonBinaryExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onBinaryExpr1(stack["partials"], stack["right"])
}

func (c *current) onPartialBinaryExprs2(first, rest interface{}) (interface{}, error) {
	return prependPartialBinaryExpr(first, rest)
}

func (p *parser) callonPartialBinaryExprs2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onPartialBinaryExprs2(stack["first"], stack["rest"])
}

func (c *current) onPartialBinaryExprs9(only interface{}) (interface{}, error) {
	return mkPartialBinaryExprs(only)
}

func (p *parser) callonPartialBinaryExprs9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onPartialBinaryExprs9(stack["only"])
}

func (c *current) onPartialBinaryExpr1(expr, op interface{}) (interface{}, error) {
	return mkPartialBinaryExpr(expr, op)
}

func (p *parser) callonPartialBinaryExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onPartialBinaryExpr1(stack["expr"], stack["op"])
}

func (c *current) onBinaryOperator1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonBinaryOperator1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onBinaryOperator1()
}

func (c *current) onUnaryExpr1(op, expr interface{}) (interface{}, error) {
	return mkUnaryExpr(op, expr)
}

func (p *parser) callonUnaryExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onUnaryExpr1(stack["op"], stack["expr"])
}

func (c *current) onUnaryOperator1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonUnaryOperator1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onUnaryOperator1()
}

func (c *current) onCall1(fn, args interface{}) (interface{}, error) {
	return mkCall(fn, args)
}

func (p *parser) callonCall1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onCall1(stack["fn"], stack["args"])
}

func (c *current) onArgs2(first, rest interface{}) (interface{}, error) {
	return prependArg(first, rest)
}

func (p *parser) callonArgs2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onArgs2(stack["first"], stack["rest"])
}

func (c *current) onArgs11(only interface{}) (interface{}, error) {
	return mkArgs(only)
}

func (p *parser) callonArgs11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onArgs11(stack["only"])
}

func (c *current) onArgs14() (interface{}, error) {
	return []expression{}, nil
}

func (p *parser) callonArgs14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onArgs14()
}

func (c *current) onLiteral1(val interface{}) (interface{}, error) {
	return val, nil
}

func (p *parser) callonLiteral1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onLiteral1(stack["val"])
}

func (c *current) onInteger1() (interface{}, error) {
	return mkInteger(c.text)
}

func (p *parser) callonInteger1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onInteger1()
}

func (c *current) onBoolean1() (interface{}, error) {
	return mkBoolean(c.text)
}

func (p *parser) callonBoolean1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onBoolean1()
}

func (c *current) onIdentifier1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonIdentifier1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onIdentifier1()
}

var (
	// errNoRule is returned when the grammar to parse has no rule.
	errNoRule = errors.New("grammar has no rule")

	// errInvalidEncoding is returned when the source is not properly
	// utf8-encoded.
	errInvalidEncoding = errors.New("invalid encoding")

	// errNoMatch is returned if no match could be found.
	errNoMatch = errors.New("no match found")
)

// Option is a function that can set an option on the parser. It returns
// the previous setting as an Option.
type Option func(*parser) Option

// Debug creates an Option to set the debug flag to b. When set to true,
// debugging information is printed to stdout while parsing.
//
// The default is false.
func Debug(b bool) Option {
	return func(p *parser) Option {
		old := p.debug
		p.debug = b
		return Debug(old)
	}
}

// Memoize creates an Option to set the memoize flag to b. When set to true,
// the parser will cache all results so each expression is evaluated only
// once. This guarantees linear parsing time even for pathological cases,
// at the expense of more memory and slower times for typical cases.
//
// The default is false.
func Memoize(b bool) Option {
	return func(p *parser) Option {
		old := p.memoize
		p.memoize = b
		return Memoize(old)
	}
}

// Recover creates an Option to set the recover flag to b. When set to
// true, this causes the parser to recover from panics and convert it
// to an error. Setting it to false can be useful while debugging to
// access the full stack trace.
//
// The default is true.
func Recover(b bool) Option {
	return func(p *parser) Option {
		old := p.recover
		p.recover = b
		return Recover(old)
	}
}

// ParseFile parses the file identified by filename.
func ParseFile(filename string, opts ...Option) (interface{}, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseReader(filename, f, opts...)
}

// ParseReader parses the data from r using filename as information in the
// error messages.
func ParseReader(filename string, r io.Reader, opts ...Option) (interface{}, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return Parse(filename, b, opts...)
}

// Parse parses the data from b using filename as information in the
// error messages.
func Parse(filename string, b []byte, opts ...Option) (interface{}, error) {
	return newParser(filename, b, opts...).parse(g)
}

// position records a position in the text.
type position struct {
	line, col, offset int
}

func (p position) String() string {
	return fmt.Sprintf("%d:%d [%d]", p.line, p.col, p.offset)
}

// savepoint stores all state required to go back to this point in the
// parser.
type savepoint struct {
	position
	rn rune
	w  int
}

type current struct {
	pos  position // start position of the match
	text []byte   // raw text of the match
}

// the AST types...

type grammar struct {
	pos   position
	rules []*rule
}

type rule struct {
	pos         position
	name        string
	displayName string
	expr        interface{}
}

type choiceExpr struct {
	pos          position
	alternatives []interface{}
}

type actionExpr struct {
	pos  position
	expr interface{}
	run  func(*parser) (interface{}, error)
}

type seqExpr struct {
	pos   position
	exprs []interface{}
}

type labeledExpr struct {
	pos   position
	label string
	expr  interface{}
}

type expr struct {
	pos  position
	expr interface{}
}

type andExpr expr
type notExpr expr
type zeroOrOneExpr expr
type zeroOrMoreExpr expr
type oneOrMoreExpr expr

type ruleRefExpr struct {
	pos  position
	name string
}

type andCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type notCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type litMatcher struct {
	pos        position
	val        string
	ignoreCase bool
}

type charClassMatcher struct {
	pos        position
	val        string
	chars      []rune
	ranges     []rune
	classes    []*unicode.RangeTable
	ignoreCase bool
	inverted   bool
}

type anyMatcher position

// errList cumulates the errors found by the parser.
type errList []error

func (e *errList) add(err error) {
	*e = append(*e, err)
}

func (e errList) err() error {
	if len(e) == 0 {
		return nil
	}
	e.dedupe()
	return e
}

func (e *errList) dedupe() {
	var cleaned []error
	set := make(map[string]bool)
	for _, err := range *e {
		if msg := err.Error(); !set[msg] {
			set[msg] = true
			cleaned = append(cleaned, err)
		}
	}
	*e = cleaned
}

func (e errList) Error() string {
	switch len(e) {
	case 0:
		return ""
	case 1:
		return e[0].Error()
	default:
		var buf bytes.Buffer

		for i, err := range e {
			if i > 0 {
				buf.WriteRune('\n')
			}
			buf.WriteString(err.Error())
		}
		return buf.String()
	}
}

// parserError wraps an error with a prefix indicating the rule in which
// the error occurred. The original error is stored in the Inner field.
type parserError struct {
	Inner  error
	pos    position
	prefix string
}

// Error returns the error message.
func (p *parserError) Error() string {
	return p.prefix + ": " + p.Inner.Error()
}

// newParser creates a parser with the specified input source and options.
func newParser(filename string, b []byte, opts ...Option) *parser {
	p := &parser{
		filename: filename,
		errs:     new(errList),
		data:     b,
		pt:       savepoint{position: position{line: 1}},
		recover:  true,
	}
	p.setOptions(opts)
	return p
}

// setOptions applies the options to the parser.
func (p *parser) setOptions(opts []Option) {
	for _, opt := range opts {
		opt(p)
	}
}

type resultTuple struct {
	v   interface{}
	b   bool
	end savepoint
}

type parser struct {
	filename string
	pt       savepoint
	cur      current

	data []byte
	errs *errList

	recover bool
	debug   bool
	depth   int

	memoize bool
	// memoization table for the packrat algorithm:
	// map[offset in source] map[expression or rule] {value, match}
	memo map[int]map[interface{}]resultTuple

	// rules table, maps the rule identifier to the rule node
	rules map[string]*rule
	// variables stack, map of label to value
	vstack []map[string]interface{}
	// rule stack, allows identification of the current rule in errors
	rstack []*rule

	// stats
	exprCnt int
}

// push a variable set on the vstack.
func (p *parser) pushV() {
	if cap(p.vstack) == len(p.vstack) {
		// create new empty slot in the stack
		p.vstack = append(p.vstack, nil)
	} else {
		// slice to 1 more
		p.vstack = p.vstack[:len(p.vstack)+1]
	}

	// get the last args set
	m := p.vstack[len(p.vstack)-1]
	if m != nil && len(m) == 0 {
		// empty map, all good
		return
	}

	m = make(map[string]interface{})
	p.vstack[len(p.vstack)-1] = m
}

// pop a variable set from the vstack.
func (p *parser) popV() {
	// if the map is not empty, clear it
	m := p.vstack[len(p.vstack)-1]
	if len(m) > 0 {
		// GC that map
		p.vstack[len(p.vstack)-1] = nil
	}
	p.vstack = p.vstack[:len(p.vstack)-1]
}

func (p *parser) print(prefix, s string) string {
	if !p.debug {
		return s
	}

	fmt.Printf("%s %d:%d:%d: %s [%#U]\n",
		prefix, p.pt.line, p.pt.col, p.pt.offset, s, p.pt.rn)
	return s
}

func (p *parser) in(s string) string {
	p.depth++
	return p.print(strings.Repeat(" ", p.depth)+">", s)
}

func (p *parser) out(s string) string {
	p.depth--
	return p.print(strings.Repeat(" ", p.depth)+"<", s)
}

func (p *parser) addErr(err error) {
	p.addErrAt(err, p.pt.position)
}

func (p *parser) addErrAt(err error, pos position) {
	var buf bytes.Buffer
	if p.filename != "" {
		buf.WriteString(p.filename)
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprintf("%d:%d (%d)", pos.line, pos.col, pos.offset))
	if len(p.rstack) > 0 {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
		rule := p.rstack[len(p.rstack)-1]
		if rule.displayName != "" {
			buf.WriteString("rule " + rule.displayName)
		} else {
			buf.WriteString("rule " + rule.name)
		}
	}
	pe := &parserError{Inner: err, pos: pos, prefix: buf.String()}
	p.errs.add(pe)
}

// read advances the parser to the next rune.
func (p *parser) read() {
	p.pt.offset += p.pt.w
	rn, n := utf8.DecodeRune(p.data[p.pt.offset:])
	p.pt.rn = rn
	p.pt.w = n
	p.pt.col++
	if rn == '\n' {
		p.pt.line++
		p.pt.col = 0
	}

	if rn == utf8.RuneError {
		if n == 1 {
			p.addErr(errInvalidEncoding)
		}
	}
}

// restore parser position to the savepoint pt.
func (p *parser) restore(pt savepoint) {
	if p.debug {
		defer p.out(p.in("restore"))
	}
	if pt.offset == p.pt.offset {
		return
	}
	p.pt = pt
}

// get the slice of bytes from the savepoint start to the current position.
func (p *parser) sliceFrom(start savepoint) []byte {
	return p.data[start.position.offset:p.pt.position.offset]
}

func (p *parser) getMemoized(node interface{}) (resultTuple, bool) {
	if len(p.memo) == 0 {
		return resultTuple{}, false
	}
	m := p.memo[p.pt.offset]
	if len(m) == 0 {
		return resultTuple{}, false
	}
	res, ok := m[node]
	return res, ok
}

func (p *parser) setMemoized(pt savepoint, node interface{}, tuple resultTuple) {
	if p.memo == nil {
		p.memo = make(map[int]map[interface{}]resultTuple)
	}
	m := p.memo[pt.offset]
	if m == nil {
		m = make(map[interface{}]resultTuple)
		p.memo[pt.offset] = m
	}
	m[node] = tuple
}

func (p *parser) buildRulesTable(g *grammar) {
	p.rules = make(map[string]*rule, len(g.rules))
	for _, r := range g.rules {
		p.rules[r.name] = r
	}
}

func (p *parser) parse(g *grammar) (val interface{}, err error) {
	if len(g.rules) == 0 {
		p.addErr(errNoRule)
		return nil, p.errs.err()
	}

	// TODO : not super critical but this could be generated
	p.buildRulesTable(g)

	if p.recover {
		// panic can be used in action code to stop parsing immediately
		// and return the panic as an error.
		defer func() {
			if e := recover(); e != nil {
				if p.debug {
					defer p.out(p.in("panic handler"))
				}
				val = nil
				switch e := e.(type) {
				case error:
					p.addErr(e)
				default:
					p.addErr(fmt.Errorf("%v", e))
				}
				err = p.errs.err()
			}
		}()
	}

	// start rule is rule [0]
	p.read() // advance to first rune
	val, ok := p.parseRule(g.rules[0])
	if !ok {
		if len(*p.errs) == 0 {
			// make sure this doesn't go out silently
			p.addErr(errNoMatch)
		}
		return nil, p.errs.err()
	}
	return val, p.errs.err()
}

func (p *parser) parseRule(rule *rule) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseRule " + rule.name))
	}

	if p.memoize {
		res, ok := p.getMemoized(rule)
		if ok {
			p.restore(res.end)
			return res.v, res.b
		}
	}

	start := p.pt
	p.rstack = append(p.rstack, rule)
	p.pushV()
	val, ok := p.parseExpr(rule.expr)
	p.popV()
	p.rstack = p.rstack[:len(p.rstack)-1]
	if ok && p.debug {
		p.print(strings.Repeat(" ", p.depth)+"MATCH", string(p.sliceFrom(start)))
	}

	if p.memoize {
		p.setMemoized(start, rule, resultTuple{val, ok, p.pt})
	}
	return val, ok
}

func (p *parser) parseExpr(expr interface{}) (interface{}, bool) {
	var pt savepoint
	var ok bool

	if p.memoize {
		res, ok := p.getMemoized(expr)
		if ok {
			p.restore(res.end)
			return res.v, res.b
		}
		pt = p.pt
	}

	p.exprCnt++
	var val interface{}
	switch expr := expr.(type) {
	case *actionExpr:
		val, ok = p.parseActionExpr(expr)
	case *andCodeExpr:
		val, ok = p.parseAndCodeExpr(expr)
	case *andExpr:
		val, ok = p.parseAndExpr(expr)
	case *anyMatcher:
		val, ok = p.parseAnyMatcher(expr)
	case *charClassMatcher:
		val, ok = p.parseCharClassMatcher(expr)
	case *choiceExpr:
		val, ok = p.parseChoiceExpr(expr)
	case *labeledExpr:
		val, ok = p.parseLabeledExpr(expr)
	case *litMatcher:
		val, ok = p.parseLitMatcher(expr)
	case *notCodeExpr:
		val, ok = p.parseNotCodeExpr(expr)
	case *notExpr:
		val, ok = p.parseNotExpr(expr)
	case *oneOrMoreExpr:
		val, ok = p.parseOneOrMoreExpr(expr)
	case *ruleRefExpr:
		val, ok = p.parseRuleRefExpr(expr)
	case *seqExpr:
		val, ok = p.parseSeqExpr(expr)
	case *zeroOrMoreExpr:
		val, ok = p.parseZeroOrMoreExpr(expr)
	case *zeroOrOneExpr:
		val, ok = p.parseZeroOrOneExpr(expr)
	default:
		panic(fmt.Sprintf("unknown expression type %T", expr))
	}
	if p.memoize {
		p.setMemoized(pt, expr, resultTuple{val, ok, p.pt})
	}
	return val, ok
}

func (p *parser) parseActionExpr(act *actionExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseActionExpr"))
	}

	start := p.pt
	val, ok := p.parseExpr(act.expr)
	if ok {
		p.cur.pos = start.position
		p.cur.text = p.sliceFrom(start)
		actVal, err := act.run(p)
		if err != nil {
			p.addErrAt(err, start.position)
		}
		val = actVal
	}
	if ok && p.debug {
		p.print(strings.Repeat(" ", p.depth)+"MATCH", string(p.sliceFrom(start)))
	}
	return val, ok
}

func (p *parser) parseAndCodeExpr(and *andCodeExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAndCodeExpr"))
	}

	ok, err := and.run(p)
	if err != nil {
		p.addErr(err)
	}
	return nil, ok
}

func (p *parser) parseAndExpr(and *andExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAndExpr"))
	}

	pt := p.pt
	p.pushV()
	_, ok := p.parseExpr(and.expr)
	p.popV()
	p.restore(pt)
	return nil, ok
}

func (p *parser) parseAnyMatcher(any *anyMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAnyMatcher"))
	}

	if p.pt.rn != utf8.RuneError {
		start := p.pt
		p.read()
		return p.sliceFrom(start), true
	}
	return nil, false
}

func (p *parser) parseCharClassMatcher(chr *charClassMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseCharClassMatcher"))
	}

	cur := p.pt.rn
	// can't match EOF
	if cur == utf8.RuneError {
		return nil, false
	}
	start := p.pt
	if chr.ignoreCase {
		cur = unicode.ToLower(cur)
	}

	// try to match in the list of available chars
	for _, rn := range chr.chars {
		if rn == cur {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of ranges
	for i := 0; i < len(chr.ranges); i += 2 {
		if cur >= chr.ranges[i] && cur <= chr.ranges[i+1] {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of Unicode classes
	for _, cl := range chr.classes {
		if unicode.Is(cl, cur) {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	if chr.inverted {
		p.read()
		return p.sliceFrom(start), true
	}
	return nil, false
}

func (p *parser) parseChoiceExpr(ch *choiceExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseChoiceExpr"))
	}

	for _, alt := range ch.alternatives {
		p.pushV()
		val, ok := p.parseExpr(alt)
		p.popV()
		if ok {
			return val, ok
		}
	}
	return nil, false
}

func (p *parser) parseLabeledExpr(lab *labeledExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseLabeledExpr"))
	}

	p.pushV()
	val, ok := p.parseExpr(lab.expr)
	p.popV()
	if ok && lab.label != "" {
		m := p.vstack[len(p.vstack)-1]
		m[lab.label] = val
	}
	return val, ok
}

func (p *parser) parseLitMatcher(lit *litMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseLitMatcher"))
	}

	start := p.pt
	for _, want := range lit.val {
		cur := p.pt.rn
		if lit.ignoreCase {
			cur = unicode.ToLower(cur)
		}
		if cur != want {
			p.restore(start)
			return nil, false
		}
		p.read()
	}
	return p.sliceFrom(start), true
}

func (p *parser) parseNotCodeExpr(not *notCodeExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseNotCodeExpr"))
	}

	ok, err := not.run(p)
	if err != nil {
		p.addErr(err)
	}
	return nil, !ok
}

func (p *parser) parseNotExpr(not *notExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseNotExpr"))
	}

	pt := p.pt
	p.pushV()
	_, ok := p.parseExpr(not.expr)
	p.popV()
	p.restore(pt)
	return nil, !ok
}

func (p *parser) parseOneOrMoreExpr(expr *oneOrMoreExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseOneOrMoreExpr"))
	}

	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			if len(vals) == 0 {
				// did not match once, no match
				return nil, false
			}
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseRuleRefExpr(ref *ruleRefExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseRuleRefExpr " + ref.name))
	}

	if ref.name == "" {
		panic(fmt.Sprintf("%s: invalid rule: missing name", ref.pos))
	}

	rule := p.rules[ref.name]
	if rule == nil {
		p.addErr(fmt.Errorf("undefined rule: %s", ref.name))
		return nil, false
	}
	return p.parseRule(rule)
}

func (p *parser) parseSeqExpr(seq *seqExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseSeqExpr"))
	}

	var vals []interface{}

	pt := p.pt
	for _, expr := range seq.exprs {
		val, ok := p.parseExpr(expr)
		if !ok {
			p.restore(pt)
			return nil, false
		}
		vals = append(vals, val)
	}
	return vals, true
}

func (p *parser) parseZeroOrMoreExpr(expr *zeroOrMoreExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseZeroOrMoreExpr"))
	}

	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseZeroOrOneExpr(expr *zeroOrOneExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseZeroOrOneExpr"))
	}

	p.pushV()
	val, _ := p.parseExpr(expr.expr)
	p.popV()
	// whether it matched or not, consider it a match
	return val, true
}

func rangeTable(class string) *unicode.RangeTable {
	if rt, ok := unicode.Categories[class]; ok {
		return rt
	}
	if rt, ok := unicode.Properties[class]; ok {
		return rt
	}
	if rt, ok := unicode.Scripts[class]; ok {
		return rt
	}

	// cannot happen
	panic(fmt.Sprintf("invalid Unicode class: %s", class))
}
