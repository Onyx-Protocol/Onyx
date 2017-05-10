package ivy

type builtin struct {
	name    string
	opcodes string
	args    []string
	result  string
}

var builtins = []builtin{
	{"sha3", "SHA3", []string{"String"}, "Hash"},
	{"sha256", "SHA256", []string{"String"}, "Hash"},
	{"size", "SIZE SWAP DROP", []string{""}, "Integer"},
	{"abs", "ABS", []string{"Integer"}, "Integer"},
	{"min", "MIN", []string{"Integer", "Integer"}, "Integer"},
	{"max", "MAX", []string{"Integer", "Integer"}, "Integer"},
	{"checkTxSig", "TXSIGHASH SWAP CHECKSIG", []string{"PublicKey", "Signature"}, "Boolean"},
	{"concat", "CAT", []string{"", ""}, "String"},
	{"concatpush", "CATPUSHDATA", []string{"", ""}, "String"},
	{"before", "MAXTIME GREATERTHAN", []string{"Time"}, "Boolean"},
	{"after", "MINTIME LESSTHAN", []string{"Time"}, "Boolean"},
	{"checkTxMultiSig", "", []string{"List", "List"}, "Boolean"}, // WARNING WARNING WOOP WOOP special case
}

type binaryOp struct {
	op         string
	precedence int
	opcodes    string

	// types of operands and result
	left, right, result string
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

	{"&^", 5, "INVERT AND", "", "", ""},
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

	// types of operand and result
	operand, result string
}

var unaryOps = []unaryOp{
	{"-", "NEGATE", "Integer", "Integer"},

	// not not allowed (for now?)
	// {"!", "NOT", "Boolean", "Boolean"},

	{"~", "INVERT", "", ""},
}

// properties[type] is a map from property names to their types
var properties = map[string]map[string]string{
	"Value": map[string]string{
		"assetAmount": "AssetAmount",
	},
	"Transaction": map[string]string{
		"after":  "Function",
		"before": "Function",
	},
}
