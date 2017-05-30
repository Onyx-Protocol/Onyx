package compiler

type builtin struct {
	name    string
	opcodes string
	args    []typeDesc
	result  typeDesc
}

var builtins = []builtin{
	{"sha3", "SHA3", []typeDesc{nilType}, hashType},
	{"sha256", "SHA256", []typeDesc{nilType}, hashType},
	{"size", "SIZE SWAP DROP", []typeDesc{nilType}, intType},
	{"abs", "ABS", []typeDesc{intType}, intType},
	{"min", "MIN", []typeDesc{intType, intType}, intType},
	{"max", "MAX", []typeDesc{intType, intType}, intType},
	{"checkTxSig", "TXSIGHASH SWAP CHECKSIG", []typeDesc{pubkeyType, sigType}, boolType},
	{"concat", "CAT", []typeDesc{nilType, nilType}, strType},
	{"concatpush", "CATPUSHDATA", []typeDesc{nilType, nilType}, strType},
	{"before", "MAXTIME GREATERTHAN", []typeDesc{timeType}, boolType},
	{"after", "MINTIME LESSTHAN", []typeDesc{timeType}, boolType},
	{"checkTxMultiSig", "", []typeDesc{listType, listType}, boolType}, // WARNING WARNING WOOP WOOP special case
}

type binaryOp struct {
	op         string
	precedence int
	opcodes    string

	left, right, result typeDesc
}

var binaryOps = []binaryOp{
	// disjunctions disallowed (for now?)
	// {"||", 1, "BOOLOR", "Boolean", "Boolean", "Boolean"},

	// and disallow this too
	// {"&&", 2, "BOOLAND", "Boolean", "Boolean", "Boolean"},

	{">", 3, "GREATERTHAN", "Integer", "Integer", "Boolean"},
	{"<", 3, "LESSTHAN", "Integer", "Integer", "Boolean"},
	{">=", 3, "GREATERTHANOREQUAL", "Integer", "Integer", "Boolean"},
	{"<=", 3, "LESSTHANOREQUAL", "Integer", "Integer", "Boolean"},

	{"==", 3, "EQUAL", "", "", "Boolean"},
	{"!=", 3, "EQUAL NOT", "", "", "Boolean"},

	{"^", 4, "XOR", "", "", ""},
	{"|", 4, "OR", "", "", ""},

	{"+", 4, "ADD", "Integer", "Integer", "Integer"},
	{"-", 4, "SUB", "Integer", "Integer", "Integer"},

	// {"&^", 5, "INVERT AND", "", "", ""},
	{"&", 5, "AND", "", "", ""},

	{"<<", 5, "LSHIFT", "Integer", "Integer", "Integer"},
	{">>", 5, "RSHIFT", "Integer", "Integer", "Integer"},

	{"%", 5, "MOD", "Integer", "Integer", "Integer"},
	{"*", 5, "MUL", "Integer", "Integer", "Integer"},
	{"/", 5, "DIV", "Integer", "Integer", "Integer"},
}

type unaryOp struct {
	op      string
	opcodes string

	operand, result typeDesc
}

var unaryOps = []unaryOp{
	{"-", "NEGATE", "Integer", "Integer"},

	// not not allowed (for now?)
	// {"!", "NOT", "Boolean", "Boolean"},

	{"~", "INVERT", "", ""},
}
