package txdb_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/core/txbuilder"
	. "chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
)

func TestListUTXOsByAsset(t *testing.T) {
	ctx := context.Background()
	dbctx := pgtest.NewContext(t)
	store := NewStore(pg.FromContext(dbctx).(*sql.DB))
	_, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}

	projectID := assettest.CreateProjectFixture(dbctx, t, "", "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(dbctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(dbctx, t, projectID, "", nil, nil)
	assetID1 := assettest.CreateAssetFixture(dbctx, t, issuerNodeID, "", "")
	assetID2 := assettest.CreateAssetFixture(dbctx, t, issuerNodeID, "", "")
	accountID1 := assettest.CreateAccountFixture(dbctx, t, managerNodeID, "", nil)
	accountID2 := assettest.CreateAccountFixture(dbctx, t, managerNodeID, "", nil)

	tx0 := assettest.Issue(dbctx, t, assetID1, []*txbuilder.Destination{
		assettest.AccountDest(dbctx, t, accountID1, assetID1, 1),
		assettest.AccountDest(dbctx, t, accountID2, assetID1, 1),
	})
	tx1 := assettest.Issue(dbctx, t, assetID2, []*txbuilder.Destination{
		assettest.AccountDest(dbctx, t, accountID1, assetID2, 1),
		assettest.AccountDest(dbctx, t, accountID2, assetID2, 1),
	})

	tx2 := assettest.Transfer(
		dbctx,
		t,
		[]*txbuilder.Source{asset.NewAccountSource(ctx, &bc.AssetAmount{
			AssetID: assetID2,
			Amount:  1,
		}, accountID2, nil, nil)},
		[]*txbuilder.Destination{assettest.AccountDest(dbctx, t, accountID1, assetID2, 1)},
	)

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// this tx should not affect the results
	assettest.Transfer(
		dbctx,
		t,
		[]*txbuilder.Source{asset.NewAccountSource(ctx, &bc.AssetAmount{
			AssetID: assetID1,
			Amount:  1,
		}, accountID1, nil, nil)},
		[]*txbuilder.Destination{assettest.AccountDest(dbctx, t, accountID2, assetID1, 1)},
	)

	out0_0 := &state.Output{
		Outpoint: bc.Outpoint{
			Hash:  tx0.Hash,
			Index: 0,
		},
		TxOutput: bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: assetID1, Amount: 1},
			Script:      tx0.Outputs[0].Script,
			Metadata:    []byte{},
		},
	}

	out0_1 := &state.Output{
		Outpoint: bc.Outpoint{
			Hash:  tx0.Hash,
			Index: 1,
		},
		TxOutput: bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: assetID1, Amount: 1},
			Script:      tx0.Outputs[1].Script,
			Metadata:    []byte{},
		},
	}

	out1_0 := &state.Output{
		Outpoint: bc.Outpoint{
			Hash:  tx1.Hash,
			Index: 0,
		},
		TxOutput: bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: assetID2, Amount: 1},
			Script:      tx1.Outputs[0].Script,
			Metadata:    []byte{},
		},
	}

	out2_0 := &state.Output{
		Outpoint: bc.Outpoint{
			Hash:  tx2.Hash,
			Index: 0,
		},
		TxOutput: bc.TxOutput{
			AssetAmount: bc.AssetAmount{AssetID: assetID2, Amount: 1},
			Script:      tx2.Outputs[0].Script,
			Metadata:    []byte{},
		},
	}

	examples := []struct {
		assetID   bc.AssetID
		prev      string
		limit     int
		wantUTXOs []*state.Output
		wantLast  string
	}{
		{
			assetID1,
			"",
			100,
			[]*state.Output{out0_0, out0_1},
			"2-0-1",
		},
		{
			assetID2,
			"",
			100,
			[]*state.Output{out1_0, out2_0},
			"2-2-0",
		},
		{
			bc.AssetID{},
			"",
			100,
			nil,
			"",
		},
		{
			assetID1,
			"",
			1,
			[]*state.Output{out0_0},
			"2-0-0",
		},
		{
			assetID1,
			"2-0-0",
			1,
			[]*state.Output{out0_1},
			"2-0-1",
		},
		{
			assetID1,
			"2-0-1",
			1,
			nil,
			"",
		},
	}

	for i, ex := range examples {
		t.Log("Example:", i)

		gotUTXOs, gotLast, err := store.ListUTXOsByAsset(ctx, ex.assetID, ex.prev, ex.limit)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if !reflect.DeepEqual(gotUTXOs, ex.wantUTXOs) {
			gotStr, err := json.MarshalIndent(gotUTXOs, "", "  ")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			wantStr, err := json.MarshalIndent(ex.wantUTXOs, "", "  ")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			t.Errorf("txs:\ngot:\n%s\nwant:\n%s", string(gotStr), string(wantStr))
		}

		if gotLast != ex.wantLast {
			t.Errorf("last: got=%s want=%s", gotLast, ex.wantLast)
		}
	}
}
