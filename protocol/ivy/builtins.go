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
