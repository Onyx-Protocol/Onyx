package asset

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/core/query"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	reg := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()

	tags1 := map[string]interface{}{"foo": "bar"}
	rawtags1 := json.RawMessage(`{"foo": "bar"}`)
	def1 := map[string]interface{}{"baz": "bar"}
	rawdef1 := json.RawMessage(`{
  "baz": "bar"
}`)
	asset1, err := reg.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, def1, "", tags1, "")
	if err != nil {
		t.Fatal(err)
	}

	tags2 := map[string]interface{}{"foo": "baz"}
	rawtags2 := json.RawMessage(`{"foo": "baz"}`)
	asset2, err := reg.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, nil, "", tags2, "")
	if err != nil {
		t.Fatal(err)
	}

	txs := []*query.AnnotatedTx{
		{
			Inputs: []*query.AnnotatedInput{
				{AssetID: asset1.AssetID},
				{AssetID: asset2.AssetID},
				{AssetID: bc.NewAssetID([32]byte{0xba, 0xd0})},
			},
			Outputs: []*query.AnnotatedOutput{
				{AssetID: asset1.AssetID},
				{AssetID: asset2.AssetID},
				{AssetID: bc.NewAssetID([32]byte{0xba, 0xd0})},
			},
		},
	}

	empty := json.RawMessage(`{}`)
	want := []*query.AnnotatedTx{
		{
			Inputs: []*query.AnnotatedInput{
				{AssetID: asset1.AssetID, AssetTags: &rawtags1, AssetIsLocal: true, AssetDefinition: &rawdef1},
				{AssetID: asset2.AssetID, AssetTags: &rawtags2, AssetIsLocal: true, AssetDefinition: &empty},
				{AssetID: bc.NewAssetID([32]byte{0xba, 0xd0}), AssetTags: &empty, AssetDefinition: &empty},
			},
			Outputs: []*query.AnnotatedOutput{
				{AssetID: asset1.AssetID, AssetTags: &rawtags1, AssetIsLocal: true, AssetDefinition: &rawdef1},
				{AssetID: asset2.AssetID, AssetTags: &rawtags2, AssetIsLocal: true, AssetDefinition: &empty},
				{AssetID: bc.NewAssetID([32]byte{0xba, 0xd0}), AssetTags: &empty, AssetDefinition: &empty},
			},
		},
	}
	err = reg.AnnotateTxs(ctx, txs)
	if err != nil {
		t.Fatal(err)
	}
	if !testutil.DeepEqual(txs, want) {
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(txs), spew.Sdump(want))
	}
}
