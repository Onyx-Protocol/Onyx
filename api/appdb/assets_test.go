package appdb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
)

func TestAssetByID(t *testing.T) {
	const sql = sampleProjectFixture + `
		INSERT INTO issuer_nodes (id, project_id, label, keyset, key_index)
			VALUES ('ag1', 'proj-id-0', 'foo', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0);
		INSERT INTO assets (id, issuer_node_id, key_index, keyset, redeem_script, label)
		VALUES(
			'0000000000000000000000000000000000000000000000000000000000000000',
			'ag1',
			0,
			'{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}',
			decode('51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae', 'hex'),
			'foo'
		);
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		got, err := AssetByID(ctx, bc.AssetID{})
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		redeem, _ := hex.DecodeString("51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae")
		key, _ := hdkey.NewXKey("xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd")
		want := &Asset{
			Hash:         bc.AssetID{},
			GroupID:      "ag1",
			AGIndex:      []uint32{0, 0},
			AIndex:       []uint32{0, 0},
			RedeemScript: redeem,
			Keys:         []*hdkey.XKey{key},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got asset = %+v want %+v", got, want)
		}

		// missing asset id
		_, err = AssetByID(ctx, bc.AssetID{1})
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("got error = %v want %v", errors.Root(err), pg.ErrUserInputNotFound)
		}
	})
}

func TestListAssets(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO issuer_nodes
			(id, project_id, key_index, keyset, label)
		VALUES
			('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0'),
			('ag-id-1', 'proj-id-0', 1, '{}', 'ag-1');
		INSERT INTO assets
			(id, issuer_node_id, key_index, redeem_script, label, sort_id)
		VALUES
			('asset-id-0', 'ag-id-0', 0, '{}', 'asset-0', 'asset0'),
			('asset-id-1', 'ag-id-0', 1, '{}', 'asset-1', 'asset1'),
			('asset-id-2', 'ag-id-1', 2, '{}', 'asset-2', 'asset2');
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			groupID string
			prev    string
			limit   int
			want    []*AssetResponse
		}{
			{
				"ag-id-0",
				"",
				5,
				[]*AssetResponse{
					{ID: "asset-id-1", Label: "asset-1"},
					{ID: "asset-id-0", Label: "asset-0"},
				},
			},
			{
				"ag-id-1",
				"",
				5,
				[]*AssetResponse{
					{ID: "asset-id-2", Label: "asset-2"},
				},
			},
			{
				"ag-id-0",
				"",
				1,
				[]*AssetResponse{
					{ID: "asset-id-1", Label: "asset-1"},
				},
			},
			{
				"ag-id-0",
				"asset1",
				5,
				[]*AssetResponse{
					{ID: "asset-id-0", Label: "asset-0"},
				},
			},
			{
				"ag-id-0",
				"asset0",
				5,
				nil,
			},
		}

		for _, ex := range examples {
			t.Logf("ListAssets(%s, %s, %d)", ex.groupID, ex.prev, ex.limit)

			got, _, err := ListAssets(ctx, ex.groupID, ex.prev, ex.limit)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("got:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestGetAsset(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label)
			VALUES ('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label, issued)
			VALUES ('asset-id-0', 'ag-id-0', 0, '{}', 'asset-0', 58);
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		got, err := GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		want := &AssetResponse{"asset-id-0", "asset-0", 58}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetAsset(%s) = %+v want %+v", "asset-id-0", got, want)
		}

		_, err = GetAsset(ctx, "nonexistent")
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("GetAsset(%s) error = %q want %q", "nonexistent", errors.Root(err), pg.ErrUserInputNotFound)
		}
	})
}

func TestAddIssuance(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('proj1', 'proj1');
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label)
			VALUES ('ag0', 'proj1', 0, '{}', 'ag0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label)
			VALUES ('a0', 'ag0', 0, '{}', 'foo');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	const q = `SELECT issued FROM assets WHERE id='a0'`
	var gotIssued, wantIssued int64

	// Test first issuance, and second
	for i := 0; i < 2; i++ {
		err := AddIssuance(ctx, "a0", 10)
		if err != nil {
			t.Fatal(err)
		}
		wantIssued += 10

		err = dbtx.QueryRow(q).Scan(&gotIssued)
		if err != nil {
			t.Fatal(err)
		}

		if gotIssued != wantIssued {
			t.Errorf("got issued = %d want %d", gotIssued, wantIssued)
		}
	}
}

func TestUpdateAsset(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label)
			VALUES ('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label, issued)
			VALUES ('asset-id-0', 'ag-id-0', 0, '{}', 'asset-0', 58);
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		assetResponse, err := GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		newLabel := "bar"
		err = UpdateAsset(ctx, assetResponse.ID, &newLabel)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		assetResponse, err = GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Fatalf("could not get asset with id asset-id-0: %v", err)
		}
		if assetResponse.Label != newLabel {
			t.Errorf("expected %s, got %s", newLabel, assetResponse.Label)
		}
	})
}

// Test that calling UpdateAsset with no new label is a no-op.
func TestUpdateAssetNoUpdate(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label)
			VALUES ('ag-id-0', 'proj-id-0', 0, '{}', 'ag-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, label, issued)
			VALUES ('asset-id-0', 'ag-id-0', 0, '{}', 'asset-0', 58);
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		assetResponse, err := GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		err = UpdateAsset(ctx, assetResponse.ID, nil)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		assetResponse, err = GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Fatalf("could not get asset with id asset-id-0: %v", err)
		}
		if assetResponse.Label != "asset-0" {
			t.Errorf("expected asset-0, got %s", assetResponse.Label)
		}
	})
}

func TestDeleteAsset(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		asset := newTestAsset(t, ctx, nil)
		assetID := asset.Hash.String()
		_, err := GetAsset(ctx, assetID)
		if err != nil {
			t.Errorf("could not get test asset with id %s: %v", assetID, err)
		}
		err = DeleteAsset(ctx, assetID)
		if err != nil {
			t.Errorf("could not delete asset with asset id %s: %v", assetID, err)
		}
		_, err = GetAsset(ctx, assetID)
		if err == nil { // sic
			t.Errorf("expected asset %s would be deleted, but it wasn't", assetID)
		} else {
			rootErr := errors.Root(err)
			if rootErr != pg.ErrUserInputNotFound {
				t.Errorf("unexpected error when trying to get deleted asset %s: %v", assetID, err)
			}
		}
	})
}
