package validation

import (
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestNoUpdateEmptyAD(t *testing.T) {
	ctx := context.Background()
	view := newTestView()
	tx := bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{{
		SignatureScript: []byte("foo"),
		Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
	}}})
	ApplyTx(ctx, view, tx)
	if len(view.adps) > 0 {
		// If metadata field is empty, no update of ADP takes place.
		// See https://github.com/chain-engineering/fedchain/blob/master/documentation/fedchain-specification.md#extract-asset-definition.
		t.Fatal("apply tx should not save an empty asset definition")
	}
}

func BenchmarkValidateTx(b *testing.B) {
	ctx := context.Background()
	view := newTestView()
	tx := txFromHex("0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31")
	ts := uint64(time.Now().Unix())
	var prevHash bc.Hash
	for i := 0; i < b.N; i++ {
		ValidateTx(ctx, view, tx, ts, &prevHash)
	}
}

type testView struct {
	outs map[bc.Outpoint]*state.Output
	adps map[bc.AssetID]*bc.AssetDefinitionPointer
}

func newTestView() *testView {
	return &testView{
		outs: make(map[bc.Outpoint]*state.Output),
		adps: make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *testView) AssetDefinitionPointer(id bc.AssetID) *bc.AssetDefinitionPointer {
	return v.adps[id]
}
func (v *testView) Output(context.Context, bc.Outpoint) *state.Output {
	return nil
}

func (v *testView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.adps[adp.AssetID] = adp
}

func (v *testView) SaveOutput(*state.Output) {}

func txFromHex(s string) *bc.Tx {
	tx := new(bc.Tx)
	err := tx.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return tx
}
