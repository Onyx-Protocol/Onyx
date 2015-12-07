package validation

import (
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func BenchmarkValidateTx(b *testing.B) {
	ctx := context.Background()
	view := &testView{make(map[bc.Outpoint]*state.Output)}
	tx := txFromHex("0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31")
	ts := uint64(time.Now().Unix())
	var prevHash bc.Hash
	for i := 0; i < b.N; i++ {
		ValidateTx(ctx, view, tx, ts, &prevHash)
	}
}

type testView struct {
	outs map[bc.Outpoint]*state.Output
}

func (v *testView) UnspentP2COutputs(context.Context, bc.ContractHash, bc.AssetID) []*state.Output {
	return nil
}

func (v *testView) AssetDefinitionPointer(bc.AssetID) *bc.AssetDefinitionPointer {
	return nil
}
func (v *testView) Output(context.Context, bc.Outpoint) *state.Output {
	return nil
}

func (v *testView) SaveAssetDefinitionPointer(*bc.AssetDefinitionPointer) {}
func (v *testView) SaveOutput(*state.Output)                              {}

func txFromHex(s string) *bc.Tx {
	tx := new(bc.Tx)
	err := tx.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return tx
}
