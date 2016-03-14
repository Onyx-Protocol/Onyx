package validation

import (
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/testutil"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestNoUpdateEmptyAD(t *testing.T) {
	ctx := context.Background()
	view := state.NewMemView(nil)
	tx := bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{{
		SignatureScript: []byte("foo"),
		Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
	}}})
	ApplyTx(ctx, view, tx)
	if len(view.Assets) > 0 {
		// If metadata field is empty, no update of ADP takes place.
		// See https://github.com/chain-engineering/fedchain/blob/master/documentation/fedchain-specification.md#extract-asset-definition.
		t.Fatal("apply tx should not save an empty asset definition")
	}
}

func TestDestructionTracking(t *testing.T) {
	ctx := context.Background()
	view := state.NewMemView(nil)
	aid := [32]byte{1}
	tx := bc.NewTx(bc.TxData{Outputs: []*bc.TxOutput{
		{
			AssetAmount: bc.AssetAmount{AssetID: aid, Amount: 5},
			Script:      []byte{txscript.OP_RETURN},
		},
	}})
	err := ApplyTx(ctx, view, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if len(view.Assets) == 0 {
		t.Fatal("no destruction was tracked")
	}

	var want uint64 = 5
	if got := view.Assets[aid].Destroyed; got != want {
		t.Fatalf("got destroyed = %d want %d", got, want)
	}

	if len(view.Outs) > 0 {
		t.Fatal("utxo should not be saved")
	}
}

func BenchmarkValidateTx(b *testing.B) {
	ctx := context.Background()
	view := state.NewMemView(nil)
	tx := txFromHex("0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31")
	ts := uint64(time.Now().Unix())
	for i := 0; i < b.N; i++ {
		ValidateTx(ctx, view, tx, ts)
	}
}

func txFromHex(s string) *bc.Tx {
	tx := new(bc.Tx)
	err := tx.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return tx
}
