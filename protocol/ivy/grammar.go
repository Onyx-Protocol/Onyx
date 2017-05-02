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
						&litMatcher{
							pos:        position{line: 5, col: 13, offset: 29},
							val:        "contract",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 24, offset: 40},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 27, offset: 43},
							label: "name",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 32, offset: 48},
								name: "Identifier",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 43, offset: 59},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 45, offset: 61},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 49, offset: 65},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 51, offset: 67},
							label: "params",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 58, offset: 74},
								name: "Params",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 65, offset: 81},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 67, offset: 83},
							val:        ")",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 71, offset: 87},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 73, offset: 89},
							val:        "{",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 77, offset: 93},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 5, col: 79, offset: 95},
							label: "clauses",
							expr: &ruleRefExpr{
								pos:  position{line: 5, col: 87, offset: 103},
								name: "Clauses",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 5, col: 95, offset: 111},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 5, col: 97, offset: 113},
							val:        "}",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Params",
			pos:  position{line: 9, col: 1, offset: 166},
			expr: &choiceExpr{
				pos: position{line: 9, col: 10, offset: 175},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 9, col: 10, offset: 175},
						run: (*parser).callonParams2,
						expr: &seqExpr{
							pos: position{line: 9, col: 11, offset: 176},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 9, col: 11, offset: 176},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 9, col: 17, offset: 182},
										name: "Params1Type",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 9, col: 29, offset: 194},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 9, col: 31, offset: 196},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 9, col: 35, offset: 200},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 9, col: 37, offset: 202},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 9, col: 42, offset: 207},
										name: "Params",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 11, col: 5, offset: 257},
						run: (*parser).callonParams11,
						expr: &labeledExpr{
							pos:   position{line: 11, col: 5, offset: 257},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 11, col: 10, offset: 262},
								name: "Params1Type",
							},
						},
					},
					&actionExpr{
						pos: position{line: 13, col: 5, offset: 299},
						run: (*parser).callonParams14,
						expr: &ruleRefExpr{
							pos:  position{line: 13, col: 5, offset: 299},
							name: "Nothing",
						},
					},
				},
			},
		},
		{
			name: "Params1Type",
			pos:  position{line: 17, col: 1, offset: 337},
			expr: &choiceExpr{
				pos: position{line: 17, col: 15, offset: 351},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 17, col: 15, offset: 351},
						run: (*parser).callonParams1Type2,
						expr: &seqExpr{
							pos: position{line: 17, col: 16, offset: 352},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 17, col: 16, offset: 352},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 17, col: 22, offset: 358},
										name: "Identifier",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 17, col: 33, offset: 369},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 17, col: 35, offset: 371},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 17, col: 39, offset: 375},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 17, col: 41, offset: 377},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 17, col: 46, offset: 382},
										name: "Params1Type",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 19, col: 5, offset: 432},
						run: (*parser).callonParams1Type11,
						expr: &seqExpr{
							pos: position{line: 19, col: 6, offset: 433},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 19, col: 6, offset: 433},
									label: "only",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 11, offset: 438},
										name: "Identifier",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 19, col: 22, offset: 449},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 19, col: 24, offset: 451},
									val:        ":",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 19, col: 28, offset: 455},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 19, col: 30, offset: 457},
									label: "typ",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 34, offset: 461},
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
			pos:  position{line: 23, col: 1, offset: 500},
			expr: &actionExpr{
				pos: position{line: 23, col: 8, offset: 507},
				run: (*parser).callonType1,
				expr: &choiceExpr{
					pos: position{line: 23, col: 9, offset: 508},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 23, col: 9, offset: 508},
							val:        "String",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 20, offset: 519},
							val:        "Integer",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 32, offset: 531},
							val:        "Hash",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 41, offset: 540},
							val:        "AssetAmount",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 57, offset: 556},
							val:        "Program",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 23, col: 69, offset: 568},
							val:        "Value",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Clauses",
			pos:  position{line: 27, col: 1, offset: 611},
			expr: &choiceExpr{
				pos: position{line: 27, col: 11, offset: 621},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 27, col: 11, offset: 621},
						run: (*parser).callonClauses2,
						expr: &seqExpr{
							pos: position{line: 27, col: 12, offset: 622},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 27, col: 12, offset: 622},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 27, col: 18, offset: 628},
										name: "Clause",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 27, col: 25, offset: 635},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 27, col: 27, offset: 637},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 27, col: 32, offset: 642},
										name: "Clauses",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 29, col: 5, offset: 693},
						run: (*parser).callonClauses9,
						expr: &labeledExpr{
							pos:   position{line: 29, col: 5, offset: 693},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 29, col: 10, offset: 698},
								name: "Clause",
							},
						},
					},
				},
			},
		},
		{
			name: "Clause",
			pos:  position{line: 33, col: 1, offset: 735},
			expr: &actionExpr{
				pos: position{line: 33, col: 10, offset: 744},
				run: (*parser).callonClause1,
				expr: &seqExpr{
					pos: position{line: 33, col: 11, offset: 745},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 33, col: 11, offset: 745},
							val:        "clause",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 20, offset: 754},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 23, offset: 757},
							label: "name",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 28, offset: 762},
								name: "Identifier",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 39, offset: 773},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 41, offset: 775},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 45, offset: 779},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 47, offset: 781},
							label: "params",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 54, offset: 788},
								name: "Params",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 61, offset: 795},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 63, offset: 797},
							val:        ")",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 67, offset: 801},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 69, offset: 803},
							val:        "{",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 73, offset: 807},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 33, col: 75, offset: 809},
							label: "statements",
							expr: &ruleRefExpr{
								pos:  position{line: 33, col: 86, offset: 820},
								name: "Statements",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 33, col: 97, offset: 831},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 33, col: 99, offset: 833},
							val:        "}",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Statements",
			pos:  position{line: 37, col: 1, offset: 887},
			expr: &choiceExpr{
				pos: position{line: 37, col: 14, offset: 900},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 37, col: 14, offset: 900},
						run: (*parser).callonStatements2,
						expr: &seqExpr{
							pos: position{line: 37, col: 15, offset: 901},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 37, col: 15, offset: 901},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 37, col: 21, offset: 907},
										name: "Statement",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 37, col: 31, offset: 917},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 37, col: 33, offset: 919},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 37, col: 38, offset: 924},
										name: "Statements",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 39, col: 5, offset: 981},
						run: (*parser).callonStatements9,
						expr: &labeledExpr{
							pos:   position{line: 39, col: 5, offset: 981},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 10, offset: 986},
								name: "Statement",
							},
						},
					},
				},
			},
		},
		{
			name: "Statement",
			pos:  position{line: 43, col: 1, offset: 1029},
			expr: &actionExpr{
				pos: position{line: 43, col: 13, offset: 1041},
				run: (*parser).callonStatement1,
				expr: &labeledExpr{
					pos:   position{line: 43, col: 13, offset: 1041},
					label: "s",
					expr: &choiceExpr{
						pos: position{line: 43, col: 16, offset: 1044},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 43, col: 16, offset: 1044},
								name: "VerifyStatement",
							},
							&ruleRefExpr{
								pos:  position{line: 43, col: 34, offset: 1062},
								name: "OutputStatement",
							},
							&ruleRefExpr{
								pos:  position{line: 43, col: 52, offset: 1080},
								name: "ReturnStatement",
							},
						},
					},
				},
			},
		},
		{
			name: "VerifyStatement",
			pos:  position{line: 47, col: 1, offset: 1118},
			expr: &actionExpr{
				pos: position{line: 47, col: 19, offset: 1136},
				run: (*parser).callonVerifyStatement1,
				expr: &seqExpr{
					pos: position{line: 47, col: 20, offset: 1137},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 47, col: 20, offset: 1137},
							val:        "verify",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 47, col: 29, offset: 1146},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 47, col: 32, offset: 1149},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 47, col: 37, offset: 1154},
								name: "Expr",
							},
						},
					},
				},
			},
		},
		{
			name: "OutputStatement",
			pos:  position{line: 51, col: 1, offset: 1189},
			expr: &actionExpr{
				pos: position{line: 51, col: 19, offset: 1207},
				run: (*parser).callonOutputStatement1,
				expr: &seqExpr{
					pos: position{line: 51, col: 20, offset: 1208},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 51, col: 20, offset: 1208},
							val:        "output",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 29, offset: 1217},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 51, col: 32, offset: 1220},
							label: "callExpr",
							expr: &ruleRefExpr{
								pos:  position{line: 51, col: 41, offset: 1229},
								name: "Call",
							},
						},
					},
				},
			},
		},
		{
			name: "ReturnStatement",
			pos:  position{line: 55, col: 1, offset: 1268},
			expr: &actionExpr{
				pos: position{line: 55, col: 19, offset: 1286},
				run: (*parser).callonReturnStatement1,
				expr: &seqExpr{
					pos: position{line: 55, col: 20, offset: 1287},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 55, col: 20, offset: 1287},
							val:        "return",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 55, col: 29, offset: 1296},
							name: "__",
						},
						&labeledExpr{
							pos:   position{line: 55, col: 32, offset: 1299},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 55, col: 37, offset: 1304},
								name: "Expr",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr",
			pos:  position{line: 59, col: 1, offset: 1339},
			expr: &actionExpr{
				pos: position{line: 59, col: 8, offset: 1346},
				run: (*parser).callonExpr1,
				expr: &labeledExpr{
					pos:   position{line: 59, col: 8, offset: 1346},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 59, col: 11, offset: 1349},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 59, col: 11, offset: 1349},
								name: "ComparisonExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 59, col: 28, offset: 1366},
								name: "Expr2",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr2",
			pos:  position{line: 63, col: 1, offset: 1394},
			expr: &actionExpr{
				pos: position{line: 63, col: 9, offset: 1402},
				run: (*parser).callonExpr21,
				expr: &labeledExpr{
					pos:   position{line: 63, col: 9, offset: 1402},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 63, col: 12, offset: 1405},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 63, col: 12, offset: 1405},
								name: "BinaryExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 63, col: 25, offset: 1418},
								name: "Expr3",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr3",
			pos:  position{line: 67, col: 1, offset: 1446},
			expr: &actionExpr{
				pos: position{line: 67, col: 9, offset: 1454},
				run: (*parser).callonExpr31,
				expr: &labeledExpr{
					pos:   position{line: 67, col: 9, offset: 1454},
					label: "e",
					expr: &choiceExpr{
						pos: position{line: 67, col: 12, offset: 1457},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 67, col: 12, offset: 1457},
								name: "UnaryExpr",
							},
							&ruleRefExpr{
								pos:  position{line: 67, col: 24, offset: 1469},
								name: "Expr4",
							},
						},
					},
				},
			},
		},
		{
			name: "Expr4",
			pos:  position{line: 71, col: 1, offset: 1497},
			expr: &choiceExpr{
				pos: position{line: 71, col: 9, offset: 1505},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 71, col: 9, offset: 1505},
						run: (*parser).callonExpr42,
						expr: &labeledExpr{
							pos:   position{line: 71, col: 9, offset: 1505},
							label: "e",
							expr: &choiceExpr{
								pos: position{line: 71, col: 12, offset: 1508},
								alternatives: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 71, col: 12, offset: 1508},
										name: "Call",
									},
									&ruleRefExpr{
										pos:  position{line: 71, col: 19, offset: 1515},
										name: "Literal",
									},
									&ruleRefExpr{
										pos:  position{line: 71, col: 29, offset: 1525},
										name: "Ref",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 73, col: 5, offset: 1552},
						run: (*parser).callonExpr48,
						expr: &seqExpr{
							pos: position{line: 73, col: 6, offset: 1553},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 73, col: 6, offset: 1553},
									val:        "(",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 73, col: 10, offset: 1557},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 73, col: 12, offset: 1559},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 73, col: 14, offset: 1561},
										name: "Expr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 73, col: 19, offset: 1566},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 73, col: 21, offset: 1568},
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
			name: "Ref",
			pos:  position{line: 77, col: 1, offset: 1594},
			expr: &choiceExpr{
				pos: position{line: 77, col: 7, offset: 1600},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 77, col: 7, offset: 1600},
						run: (*parser).callonRef2,
						expr: &seqExpr{
							pos: position{line: 77, col: 8, offset: 1601},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 77, col: 8, offset: 1601},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 14, offset: 1607},
										name: "Identifier",
									},
								},
								&litMatcher{
									pos:        position{line: 77, col: 25, offset: 1618},
									val:        ".",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 77, col: 29, offset: 1622},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 34, offset: 1627},
										name: "Ref",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 79, col: 5, offset: 1671},
						run: (*parser).callonRef9,
						expr: &labeledExpr{
							pos:   position{line: 79, col: 5, offset: 1671},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 79, col: 10, offset: 1676},
								name: "Identifier",
							},
						},
					},
				},
			},
		},
		{
			name: "ComparisonExpr",
			pos:  position{line: 84, col: 1, offset: 1732},
			expr: &actionExpr{
				pos: position{line: 84, col: 18, offset: 1749},
				run: (*parser).callonComparisonExpr1,
				expr: &seqExpr{
					pos: position{line: 84, col: 19, offset: 1750},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 84, col: 19, offset: 1750},
							label: "left",
							expr: &ruleRefExpr{
								pos:  position{line: 84, col: 24, offset: 1755},
								name: "Expr2",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 84, col: 30, offset: 1761},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 84, col: 32, offset: 1763},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 84, col: 35, offset: 1766},
								name: "ComparisonOperator",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 84, col: 54, offset: 1785},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 84, col: 56, offset: 1787},
							label: "right",
							expr: &ruleRefExpr{
								pos:  position{line: 84, col: 62, offset: 1793},
								name: "Expr2",
							},
						},
					},
				},
			},
		},
		{
			name: "ComparisonOperator",
			pos:  position{line: 88, col: 1, offset: 1844},
			expr: &actionExpr{
				pos: position{line: 88, col: 22, offset: 1865},
				run: (*parser).callonComparisonOperator1,
				expr: &choiceExpr{
					pos: position{line: 88, col: 23, offset: 1866},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 88, col: 23, offset: 1866},
							val:        "==",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 88, col: 30, offset: 1873},
							val:        "!=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 88, col: 37, offset: 1880},
							val:        "<=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 88, col: 44, offset: 1887},
							val:        ">=",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 88, col: 51, offset: 1894},
							val:        "<",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 88, col: 57, offset: 1900},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "BinaryExpr",
			pos:  position{line: 93, col: 1, offset: 1959},
			expr: &actionExpr{
				pos: position{line: 93, col: 14, offset: 1972},
				run: (*parser).callonBinaryExpr1,
				expr: &seqExpr{
					pos: position{line: 93, col: 15, offset: 1973},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 93, col: 15, offset: 1973},
							label: "partials",
							expr: &ruleRefExpr{
								pos:  position{line: 93, col: 24, offset: 1982},
								name: "PartialBinaryExprs",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 93, col: 43, offset: 2001},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 93, col: 45, offset: 2003},
							label: "right",
							expr: &ruleRefExpr{
								pos:  position{line: 93, col: 51, offset: 2009},
								name: "Expr3",
							},
						},
					},
				},
			},
		},
		{
			name: "PartialBinaryExprs",
			pos:  position{line: 97, col: 1, offset: 2083},
			expr: &choiceExpr{
				pos: position{line: 97, col: 22, offset: 2104},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 97, col: 22, offset: 2104},
						run: (*parser).callonPartialBinaryExprs2,
						expr: &seqExpr{
							pos: position{line: 97, col: 23, offset: 2105},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 97, col: 23, offset: 2105},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 97, col: 29, offset: 2111},
										name: "PartialBinaryExpr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 97, col: 47, offset: 2129},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 97, col: 49, offset: 2131},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 97, col: 54, offset: 2136},
										name: "PartialBinaryExprs",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 99, col: 5, offset: 2209},
						run: (*parser).callonPartialBinaryExprs9,
						expr: &labeledExpr{
							pos:   position{line: 99, col: 5, offset: 2209},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 99, col: 10, offset: 2214},
								name: "PartialBinaryExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "PartialBinaryExpr",
			pos:  position{line: 103, col: 1, offset: 2273},
			expr: &actionExpr{
				pos: position{line: 103, col: 21, offset: 2293},
				run: (*parser).callonPartialBinaryExpr1,
				expr: &seqExpr{
					pos: position{line: 103, col: 22, offset: 2294},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 103, col: 22, offset: 2294},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 103, col: 27, offset: 2299},
								name: "Expr3",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 103, col: 33, offset: 2305},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 103, col: 35, offset: 2307},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 103, col: 38, offset: 2310},
								name: "BinaryOperator",
							},
						},
					},
				},
			},
		},
		{
			name: "BinaryOperator",
			pos:  position{line: 107, col: 1, offset: 2370},
			expr: &actionExpr{
				pos: position{line: 107, col: 18, offset: 2387},
				run: (*parser).callonBinaryOperator1,
				expr: &choiceExpr{
					pos: position{line: 107, col: 19, offset: 2388},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 107, col: 19, offset: 2388},
							val:        "+",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 107, col: 25, offset: 2394},
							val:        "-",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "UnaryExpr",
			pos:  position{line: 111, col: 1, offset: 2433},
			expr: &actionExpr{
				pos: position{line: 111, col: 13, offset: 2445},
				run: (*parser).callonUnaryExpr1,
				expr: &seqExpr{
					pos: position{line: 111, col: 14, offset: 2446},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 111, col: 14, offset: 2446},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 111, col: 17, offset: 2449},
								name: "UnaryOperator",
							},
						},
						&labeledExpr{
							pos:   position{line: 111, col: 31, offset: 2463},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 111, col: 36, offset: 2468},
								name: "Expr4",
							},
						},
					},
				},
			},
		},
		{
			name: "UnaryOperator",
			pos:  position{line: 115, col: 1, offset: 2511},
			expr: &actionExpr{
				pos: position{line: 115, col: 17, offset: 2527},
				run: (*parser).callonUnaryOperator1,
				expr: &choiceExpr{
					pos: position{line: 115, col: 18, offset: 2528},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 115, col: 18, offset: 2528},
							val:        "-",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 115, col: 24, offset: 2534},
							val:        "!",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Call",
			pos:  position{line: 119, col: 1, offset: 2573},
			expr: &actionExpr{
				pos: position{line: 119, col: 8, offset: 2580},
				run: (*parser).callonCall1,
				expr: &seqExpr{
					pos: position{line: 119, col: 9, offset: 2581},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 119, col: 9, offset: 2581},
							label: "fn",
							expr: &ruleRefExpr{
								pos:  position{line: 119, col: 12, offset: 2584},
								name: "Ref",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 119, col: 16, offset: 2588},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 119, col: 18, offset: 2590},
							val:        "(",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 119, col: 22, offset: 2594},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 119, col: 24, offset: 2596},
							label: "args",
							expr: &ruleRefExpr{
								pos:  position{line: 119, col: 29, offset: 2601},
								name: "Args",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 119, col: 34, offset: 2606},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 119, col: 36, offset: 2608},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Args",
			pos:  position{line: 123, col: 1, offset: 2644},
			expr: &choiceExpr{
				pos: position{line: 123, col: 8, offset: 2651},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 123, col: 8, offset: 2651},
						run: (*parser).callonArgs2,
						expr: &seqExpr{
							pos: position{line: 123, col: 9, offset: 2652},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 123, col: 9, offset: 2652},
									label: "first",
									expr: &ruleRefExpr{
										pos:  position{line: 123, col: 15, offset: 2658},
										name: "Expr",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 123, col: 20, offset: 2663},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 123, col: 22, offset: 2665},
									val:        ",",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 123, col: 26, offset: 2669},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 123, col: 28, offset: 2671},
									label: "rest",
									expr: &ruleRefExpr{
										pos:  position{line: 123, col: 33, offset: 2676},
										name: "Args",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 2721},
						run: (*parser).callonArgs11,
						expr: &labeledExpr{
							pos:   position{line: 125, col: 5, offset: 2721},
							label: "only",
							expr: &ruleRefExpr{
								pos:  position{line: 125, col: 10, offset: 2726},
								name: "Expr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 127, col: 5, offset: 2759},
						run: (*parser).callonArgs14,
						expr: &ruleRefExpr{
							pos:  position{line: 127, col: 5, offset: 2759},
							name: "Nothing",
						},
					},
				},
			},
		},
		{
			name: "Literal",
			pos:  position{line: 131, col: 1, offset: 2801},
			expr: &actionExpr{
				pos: position{line: 131, col: 11, offset: 2811},
				run: (*parser).callonLiteral1,
				expr: &labeledExpr{
					pos:   position{line: 131, col: 11, offset: 2811},
					label: "val",
					expr: &choiceExpr{
						pos: position{line: 131, col: 16, offset: 2816},
						alternatives: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 131, col: 16, offset: 2816},
								name: "Integer",
							},
							&ruleRefExpr{
								pos:  position{line: 131, col: 26, offset: 2826},
								name: "Boolean",
							},
						},
					},
				},
			},
		},
		{
			name: "Integer",
			pos:  position{line: 135, col: 1, offset: 2858},
			expr: &actionExpr{
				pos: position{line: 135, col: 11, offset: 2868},
				run: (*parser).callonInteger1,
				expr: &seqExpr{
					pos: position{line: 135, col: 12, offset: 2869},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 135, col: 12, offset: 2869},
							expr: &litMatcher{
								pos:        position{line: 135, col: 12, offset: 2869},
								val:        "-",
								ignoreCase: false,
							},
						},
						&oneOrMoreExpr{
							pos: position{line: 135, col: 17, offset: 2874},
							expr: &charClassMatcher{
								pos:        position{line: 135, col: 17, offset: 2874},
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
			pos:  position{line: 139, col: 1, offset: 2914},
			expr: &actionExpr{
				pos: position{line: 139, col: 11, offset: 2924},
				run: (*parser).callonBoolean1,
				expr: &choiceExpr{
					pos: position{line: 139, col: 12, offset: 2925},
					alternatives: []interface{}{
						&litMatcher{
							pos:        position{line: 139, col: 12, offset: 2925},
							val:        "true",
							ignoreCase: false,
						},
						&litMatcher{
							pos:        position{line: 139, col: 21, offset: 2934},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "Identifier",
			pos:  position{line: 143, col: 1, offset: 2975},
			expr: &actionExpr{
				pos: position{line: 143, col: 14, offset: 2988},
				run: (*parser).callonIdentifier1,
				expr: &seqExpr{
					pos: position{line: 143, col: 14, offset: 2988},
					exprs: []interface{}{
						&charClassMatcher{
							pos:        position{line: 143, col: 14, offset: 2988},
							val:        "[A-Za-z]",
							ranges:     []rune{'A', 'Z', 'a', 'z'},
							ignoreCase: false,
							inverted:   false,
						},
						&zeroOrMoreExpr{
							pos: position{line: 143, col: 22, offset: 2996},
							expr: &charClassMatcher{
								pos:        position{line: 143, col: 22, offset: 2996},
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
			pos:  position{line: 147, col: 1, offset: 3043},
			expr: &oneOrMoreExpr{
				pos: position{line: 147, col: 14, offset: 3056},
				expr: &charClassMatcher{
					pos:        position{line: 147, col: 14, offset: 3056},
					val:        "[ \\t\\n\\r]",
					chars:      []rune{' ', '\t', '\n', '\r'},
					ignoreCase: false,
					inverted:   false,
				},
			},
		},
		{
			name: "Comment",
			pos:  position{line: 149, col: 1, offset: 3068},
			expr: &seqExpr{
				pos: position{line: 149, col: 11, offset: 3078},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 149, col: 11, offset: 3078},
						val:        "#",
						ignoreCase: false,
					},
					&zeroOrMoreExpr{
						pos: position{line: 149, col: 15, offset: 3082},
						expr: &charClassMatcher{
							pos:        position{line: 149, col: 15, offset: 3082},
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
			pos:  position{line: 151, col: 1, offset: 3092},
			expr: &zeroOrMoreExpr{
				pos: position{line: 151, col: 5, offset: 3096},
				expr: &choiceExpr{
					pos: position{line: 151, col: 6, offset: 3097},
					alternatives: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 151, col: 6, offset: 3097},
							name: "Whitespace",
						},
						&ruleRefExpr{
							pos:  position{line: 151, col: 19, offset: 3110},
							name: "Comment",
						},
					},
				},
			},
		},
		{
			name: "__",
			pos:  position{line: 153, col: 1, offset: 3121},
			expr: &oneOrMoreExpr{
				pos: position{line: 153, col: 6, offset: 3126},
				expr: &choiceExpr{
					pos: position{line: 153, col: 7, offset: 3127},
					alternatives: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 153, col: 7, offset: 3127},
							name: "Whitespace",
						},
						&ruleRefExpr{
							pos:  position{line: 153, col: 20, offset: 3140},
							name: "Comment",
						},
					},
				},
			},
		},
		{
			name: "Nothing",
			pos:  position{line: 155, col: 1, offset: 3151},
			expr: &ruleRefExpr{
				pos:  position{line: 155, col: 11, offset: 3161},
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

func (c *current) onRef2(first, rest interface{}) (interface{}, error) {
	return prependRef(first, rest)
}

func (p *parser) callonRef2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onRef2(stack["first"], stack["rest"])
}

func (c *current) onRef9(only interface{}) (interface{}, error) {
	return mkRef(only)
}

func (p *parser) callonRef9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onRef9(stack["only"])
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
