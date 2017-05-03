package ivy

import "chain/protocol/vm"

type builtin struct {
	name      string
	signature signature
	ops       []byte
}

type signature struct {
	args   []string
	result string
}

var builtins = []*builtin{
	{
		name: "sha3",
		signature: signature{
			args:   []string{"String"},
			result: "Hash",
		},
		ops: []byte{byte(vm.OP_SHA3)},
	},
}

var binaryOps = map[string]signature{
	"==": signature{
		args:   []string{"", ""},
		result: "Boolean",
	},
	"!=": signature{
		args:   []string{"", ""},
		result: "Boolean",
	},
}

var unaryOps = map[string]signature{
	"!": signature{
		args:   []string{""},
		result: "Boolean",
	},
	"-": signature{
		args:   []string{"Integer"},
		result: "Integer",
	},
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
