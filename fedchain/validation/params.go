package validation

import "chain/fedchain/script"

// Params describes all parameters for consensus on a given blockchain.
type Params struct {
	GenesisHash  [32]byte
	ScriptParams script.Params
}

// TestParams provides parameters for testing.
// Any concrete block chain should specify its own parameters explicitly.
var TestParams = Params{
	ScriptParams: script.Params{
		Flags:           script.DefaultFlags,
		MaxPushdataSize: 520,
		MaxOpCount:      201,
		MaxStackSize:    1000,
		MaxScriptSize:   10000,
		IntegerMaxSize:  4,
		LockTimeMaxSize: 5,
		PanicOnFailure:  false,
	},
}
