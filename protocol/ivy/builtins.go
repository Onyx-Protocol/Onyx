package ivy

import "chain/protocol/vm"

type builtin struct {
	name    string
	opcodes []vm.Op
	args    []typeDesc
	result  typeDesc
}

var builtins = []builtin{
	{"sha3", []vm.Op{vm.OP_SHA3}, []typeDesc{nilType}, hashType},
	{"sha256", []vm.Op{vm.OP_SHA256}, []typeDesc{nilType}, hashType},
	{"size", []vm.Op{vm.OP_SIZE, vm.OP_SWAP, vm.OP_DROP}, []typeDesc{nilType}, intType},
	{"abs", []vm.Op{vm.OP_ABS}, []typeDesc{intType}, intType},
	{"min", []vm.Op{vm.OP_MIN}, []typeDesc{intType, intType}, intType},
	{"max", []vm.Op{vm.OP_MAX}, []typeDesc{intType, intType}, intType},
	{"checkTxSig", []vm.Op{vm.OP_TXSIGHASH, vm.OP_SWAP, vm.OP_CHECKSIG}, []typeDesc{pubkeyType, sigType}, boolType},
	{"concat", []vm.Op{vm.OP_CAT}, []typeDesc{nilType, nilType}, strType},
	{"concatpush", []vm.Op{vm.OP_CATPUSHDATA}, []typeDesc{nilType, nilType}, strType},
	{"before", []vm.Op{vm.OP_MAXTIME, vm.OP_GREATERTHAN}, []typeDesc{timeType}, boolType},
	{"after", []vm.Op{vm.OP_MINTIME, vm.OP_LESSTHAN}, []typeDesc{timeType}, boolType},
	{"checkTxMultiSig", nil, []typeDesc{listType, listType}, boolType}, // WARNING WARNING WOOP WOOP special case
}

type binaryOp struct {
	op         string
	precedence int
	opcodes    []vm.Op

	left, right, result typeDesc
}

var binaryOps = []binaryOp{
	// disjunctions disallowed (for now?)
	// {"||", 1, "BOOLOR", "Boolean", "Boolean", "Boolean"},

	// and disallow this too
	// {"&&", 2, "BOOLAND", "Boolean", "Boolean", "Boolean"},

	{">", 3, []vm.Op{vm.OP_GREATERTHAN}, "Integer", "Integer", "Boolean"},
	{"<", 3, []vm.Op{vm.OP_LESSTHAN}, "Integer", "Integer", "Boolean"},
	{">=", 3, []vm.Op{vm.OP_GREATERTHANOREQUAL}, "Integer", "Integer", "Boolean"},
	{"<=", 3, []vm.Op{vm.OP_LESSTHANOREQUAL}, "Integer", "Integer", "Boolean"},

	{"==", 3, []vm.Op{vm.OP_EQUAL}, "", "", "Boolean"},
	{"!=", 3, []vm.Op{vm.OP_EQUAL, vm.OP_NOT}, "", "", "Boolean"},

	{"^", 4, []vm.Op{vm.OP_XOR}, "", "", ""},
	{"|", 4, []vm.Op{vm.OP_OR}, "", "", ""},

	{"+", 4, []vm.Op{vm.OP_ADD}, "Integer", "Integer", "Integer"},
	{"-", 4, []vm.Op{vm.OP_SUB}, "Integer", "Integer", "Integer"},

	{"&^", 5, []vm.Op{vm.OP_INVERT, vm.OP_AND}, "", "", ""},
	{"&", 5, []vm.Op{vm.OP_AND}, "", "", ""},

	{"<<", 5, []vm.Op{vm.OP_LSHIFT}, "Integer", "Integer", "Integer"},
	{">>", 5, []vm.Op{vm.OP_RSHIFT}, "Integer", "Integer", "Integer"},

	{"%", 5, []vm.Op{vm.OP_MOD}, "Integer", "Integer", "Integer"},
	{"*", 5, []vm.Op{vm.OP_MUL}, "Integer", "Integer", "Integer"},
	{"/", 5, []vm.Op{vm.OP_DIV}, "Integer", "Integer", "Integer"},
}

type unaryOp struct {
	op      string
	opcodes []vm.Op

	operand, result typeDesc
}

var unaryOps = []unaryOp{
	{"-", []vm.Op{vm.OP_NEGATE}, "Integer", "Integer"},

	// not not allowed (for now?)
	// {"!", []vm.Op{vm.OP_NOT}, "Boolean", "Boolean"},

	{"~", []vm.Op{vm.OP_INVERT}, "", ""},
}
