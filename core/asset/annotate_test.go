package asset

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAnnotateTxs(t *testing.T) {
	reg := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()

	tags1 := map[string]interface{}{"foo": "bar"}
	def1 := map[string]interface{}{"baz": "bar"}

	asset1, err := reg.Define(ctx, []string{testutil.TestXPub.String()}, 1, def1, "", tags1, nil)
	if err != nil {
		t.Fatal(err)
	}

	tags2 := map[string]interface{}{"foo": "baz"}
	asset2, err := reg.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, "", tags2, nil)
	if err != nil {
		t.Fatal(err)
	}

	txs := []map[string]interface{}{
		{
			"inputs": []interface{}{
				map[string]interface{}{
					"asset_id": asset1.AssetID.String(),
				},
				map[string]interface{}{
					"asset_id": asset2.AssetID.String(),
				},
				map[string]interface{}{
					"asset_id": "unknown",
				},
			},
			"outputs": []interface{}{
				map[string]interface{}{
					"asset_id": asset1.AssetID.String(),
				},
				map[string]interface{}{
					"asset_id": asset2.AssetID.String(),
				},
				map[string]interface{}{
					"asset_id": "unknown",
				},
			},
		},
	}
	want := []map[string]interface{}{
		{
			"inputs": []interface{}{
				map[string]interface{}{
					"asset_id":         asset1.AssetID.String(),
					"asset_tags":       interface{}(tags1),
					"asset_is_local":   "yes",
					"asset_definition": interface{}(def1),
				},
				map[string]interface{}{
					"asset_id":         asset2.AssetID.String(),
					"asset_tags":       interface{}(tags2),
					"asset_is_local":   "yes",
					"asset_definition": map[string]interface{}{},
				},
				map[string]interface{}{
					"asset_id":         "unknown",
					"asset_tags":       map[string]interface{}{},
					"asset_is_local":   "no",
					"asset_definition": map[string]interface{}{},
				},
			},
			"outputs": []interface{}{
				map[string]interface{}{
					"asset_id":         asset1.AssetID.String(),
					"asset_tags":       interface{}(tags1),
					"asset_is_local":   "yes",
					"asset_definition": interface{}(def1),
				},
				map[string]interface{}{
					"asset_id":         asset2.AssetID.String(),
					"asset_tags":       interface{}(tags2),
					"asset_is_local":   "yes",
					"asset_definition": map[string]interface{}{},
				},
				map[string]interface{}{
					"asset_id":         "unknown",
					"asset_tags":       map[string]interface{}{},
					"asset_is_local":   "no",
					"asset_definition": map[string]interface{}{},
				},
			},
		},
	}

	err = reg.AnnotateTxs(ctx, txs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(txs, want) {
		t.Errorf("got:\n%+v\nwant:\n%+v", txs, want)
	}
}
