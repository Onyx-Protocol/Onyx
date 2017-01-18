package asset

import (
	"context"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/core/query"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	reg := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()

	tags1 := map[string]interface{}{"foo": "bar"}
	rawtags1 := []byte(`{"foo": "bar"}`)
	def1 := map[string]interface{}{"baz": "bar"}
	rawdef1 := []byte(`{
  "baz": "bar"
}`)
	asset1, err := reg.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, def1, "", tags1, "")
	if err != nil {
		t.Fatal(err)
	}

	tags2 := map[string]interface{}{"foo": "baz"}
	rawtags2 := []byte(`{"foo": "baz"}`)
	asset2, err := reg.Define(ctx, []chainkd.XPub{testutil.TestXPub}, 1, nil, "", tags2, "")
	if err != nil {
		t.Fatal(err)
	}

	txs := []*query.AnnotatedTx{
		{
			Inputs: []*query.AnnotatedInput{
				{AssetID: asset1.AssetID[:]},
				{AssetID: asset2.AssetID[:]},
				{AssetID: []byte{0xba, 0xd0}},
			},
			Outputs: []*query.AnnotatedOutput{
				{AssetID: asset1.AssetID[:]},
				{AssetID: asset2.AssetID[:]},
				{AssetID: []byte{0xba, 0xd0}},
			},
		},
	}
	want := []*query.AnnotatedTx{
		{
			Inputs: []*query.AnnotatedInput{
				{AssetID: asset1.AssetID[:], AssetTags: rawtags1, AssetIsLocal: true, AssetDefinition: rawdef1},
				{AssetID: asset2.AssetID[:], AssetTags: rawtags2, AssetIsLocal: true, AssetDefinition: []byte(`{}`)},
				{AssetID: []byte{0xba, 0xd0}, AssetTags: []byte(`{}`), AssetDefinition: []byte(`{}`)},
			},
			Outputs: []*query.AnnotatedOutput{
				{AssetID: asset1.AssetID[:], AssetTags: rawtags1, AssetIsLocal: true, AssetDefinition: rawdef1},
				{AssetID: asset2.AssetID[:], AssetTags: rawtags2, AssetIsLocal: true, AssetDefinition: []byte(`{}`)},
				{AssetID: []byte{0xba, 0xd0}, AssetTags: []byte(`{}`), AssetDefinition: []byte(`{}`)},
			},
		},
	}
	err = reg.AnnotateTxs(ctx, txs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(txs, want) {
		spew.Dump(txs)
		spew.Dump(want)
		t.Errorf("got:\n%+v\nwant:\n%+v", txs, want)
	}
}
