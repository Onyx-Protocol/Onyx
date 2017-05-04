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
}

var binaryOps = map[string]struct {
	opcodes             string
	left, right, result string
}{
	"||": {"BOOLOR", "Boolean", "Boolean", "Boolean"},
	"&&": {"BOOLAND", "Boolean", "Boolean", "Boolean"},

	">":  {"GREATERTHAN", "Integer", "Integer", "Boolean"},
	"<":  {"LESSTHAN", "Integer", "Integer", "Boolean"},
	">=": {"GREATERTHANOREQUAL", "Integer", "Integer", "Boolean"},
	"<=": {"LESSTHANOREQUAL", "Integer", "Integer", "Boolean"},

	"==": {"EQUAL", "", "", "Boolean"},
	"!=": {"EQUAL NOT", "", "", "Boolean"},

	"^": {"XOR", "", "", ""},
	"|": {"OR", "", "", ""},

	"+": {"ADD", "Integer", "Integer", "Integer"},
	"-": {"SUB", "Integer", "Integer", "Integer"},

	"&^": {"INVERT AND", "", "", ""},
	"&":  {"AND", "", "", ""},

	"<<": {"LSHIFT", "Integer", "Integer", "Integer"},
	">>": {"RSHIFT", "Integer", "Integer", "Integer"},

	"%": {"MOD", "Integer", "Integer", "Integer"},
	"*": {"MUL", "Integer", "Integer", "Integer"},
	"/": {"DIV", "Integer", "Integer", "Integer"},
}

var unaryOps = map[string]struct {
	opcodes         string
	operand, result string
}{
	"-": {"NEGATE", "Integer", "Integer"},
	"!": {"NOT", "Boolean", "Boolean"},
	"^": {"INVERT", "", ""},
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
