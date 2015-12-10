package appdb

import (
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
)

func TestAssetByID(t *testing.T) {
	const sql = sampleProjectFixture + `
		INSERT INTO issuer_nodes (id, project_id, label, keyset, key_index)
			VALUES ('in1', 'proj-id-0', 'foo', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0);
		INSERT INTO assets (id, issuer_node_id, key_index, keyset, redeem_script, issuance_script, label)
		VALUES(
			'0000000000000000000000000000000000000000000000000000000000000000',
			'in1',
			0,
			'{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}',
			decode('51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae', 'hex'),
			'\x'::bytea,
			'foo'
		);
	`
	withContext(t, sql, func(ctx context.Context) {
		got, err := AssetByID(ctx, bc.AssetID{})
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		redeem, _ := hex.DecodeString("51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae")
		key, _ := hdkey.NewXKey("xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd")
		want := &Asset{
			Hash:         bc.AssetID{},
			IssuerNodeID: "in1",
			INIndex:      []uint32{0, 0},
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
			('in-id-0', 'proj-id-0', 0, '{}', 'in-0'),
			('in-id-1', 'proj-id-0', 1, '{}', 'in-1');
		INSERT INTO assets
			(id, issuer_node_id, key_index, redeem_script, issuance_script, label, sort_id)
		VALUES
			('asset-id-0', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0', 'asset0'),
			('asset-id-1', 'in-id-0', 1, '\x'::bytea, '\x'::bytea, 'asset-1', 'asset1'),
			('asset-id-2', 'in-id-1', 2, '\x'::bytea, '\x'::bytea, 'asset-2', 'asset2');
		INSERT INTO issuance_totals
			(asset_id, confirmed, pool)
		VALUES
			('asset-id-0', 1, 2),
			('asset-id-1', 3, 4),
			('asset-id-2', 5, 6);
	`
	withContext(t, sql, func(ctx context.Context) {
		examples := []struct {
			inodeID string
			prev    string
			limit   int
			want    []*AssetResponse
		}{
			{
				"in-id-0",
				"",
				5,
				[]*AssetResponse{
					{ID: "asset-id-1", Label: "asset-1", Circulation: AssetCirculation{3, 7}},
					{ID: "asset-id-0", Label: "asset-0", Circulation: AssetCirculation{1, 3}},
				},
			},
			{
				"in-id-1",
				"",
				5,
				[]*AssetResponse{
					{ID: "asset-id-2", Label: "asset-2", Circulation: AssetCirculation{5, 11}},
				},
			},
			{
				"in-id-0",
				"",
				1,
				[]*AssetResponse{
					{ID: "asset-id-1", Label: "asset-1", Circulation: AssetCirculation{3, 7}},
				},
			},
			{
				"in-id-0",
				"asset1",
				5,
				[]*AssetResponse{
					{ID: "asset-id-0", Label: "asset-0", Circulation: AssetCirculation{1, 3}},
				},
			},
			{
				"in-id-0",
				"asset0",
				5,
				nil,
			},
		}

		for _, ex := range examples {
			t.Logf("ListAssets(%s, %s, %d)", ex.inodeID, ex.prev, ex.limit)

			got, _, err := ListAssets(ctx, ex.inodeID, ex.prev, ex.limit)
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
			VALUES ('in-id-0', 'proj-id-0', 0, '{}', 'in-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, issuance_script, label)
			VALUES ('asset-id-0', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0');
		INSERT INTO issuance_totals (asset_id, confirmed, pool)
			VALUES ('asset-id-0', 58, 12);
	`
	withContext(t, sql, func(ctx context.Context) {
		got, err := GetAsset(ctx, "asset-id-0")
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		want := &AssetResponse{"asset-id-0", "asset-0", AssetCirculation{58, 70}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetAsset(%s) = %+v want %+v", "asset-id-0", got, want)
		}

		_, err = GetAsset(ctx, "nonexistent")
		if errors.Root(err) != pg.ErrUserInputNotFound {
			t.Errorf("GetAsset(%s) error = %q want %q", "nonexistent", errors.Root(err), pg.ErrUserInputNotFound)
		}
	})
}

func TestUpdateIssuances(t *testing.T) {
	const fix = `
		INSERT INTO issuer_nodes (id, project_id, label, keyset, key_index)
			VALUES ('issuer-node-id-0', 'project-id-0', 'foo', '{}', 0);

		INSERT INTO assets (id, issuer_node_id, key_index, keyset, redeem_script, issuance_script, label)
		VALUES ('asset-id-0', 'issuer-node-id-0', 0, '{}', '', '', ''),
			('asset-id-1', 'issuer-node-id-0', 1, '{}', '', '', ''),
			('asset-id-2', 'issuer-node-id-0', 2, '{}', '', '', '');
		INSERT INTO issuance_totals (asset_id, confirmed, pool)
		VALUES
			('asset-id-0', 10, 10),
			('asset-id-1', 10, 10),
			('asset-id-2', 10, 10);
	`

	examples := []struct {
		deltas    map[string]int64
		confirmed bool
		want      map[string]AssetCirculation
	}{
		// Example: what happens to confirmation numbers when a block lands.
		{
			deltas: map[string]int64{
				"asset-id-0": 1,
				"asset-id-1": 2,
				"asset-id-2": 3,
			},
			confirmed: true,
			want: map[string]AssetCirculation{
				"asset-id-0": AssetCirculation{
					Confirmed: 11,
					Total:     21,
				},
				"asset-id-1": AssetCirculation{
					Confirmed: 12,
					Total:     22,
				},
				"asset-id-2": AssetCirculation{
					Confirmed: 13,
					Total:     23,
				},
			},
		},
		// Example: what happens to pool/unconfirmed numbers when a block lands.
		{
			deltas: map[string]int64{
				"asset-id-0": -1,
				"asset-id-1": -2,
				"asset-id-2": -3,
			},
			confirmed: false,
			want: map[string]AssetCirculation{
				"asset-id-0": AssetCirculation{
					Confirmed: 10,
					Total:     19,
				},
				"asset-id-1": AssetCirculation{
					Confirmed: 10,
					Total:     18,
				},
				"asset-id-2": AssetCirculation{
					Confirmed: 10,
					Total:     17,
				},
			},
		},
		// Example: what happens to pool/unconfirmed numbers when a tx lands.
		{
			deltas:    map[string]int64{"asset-id-0": 5},
			confirmed: false,
			want: map[string]AssetCirculation{
				"asset-id-0": AssetCirculation{
					Confirmed: 10,
					Total:     25,
				},
				"asset-id-1": AssetCirculation{
					Confirmed: 10,
					Total:     20,
				},
				"asset-id-2": AssetCirculation{
					Confirmed: 10,
					Total:     20,
				},
			},
		},
	}

	for i, ex := range examples {
		withContext(t, fix, func(ctx context.Context) {
			t.Log("Example", i)

			err := UpdateIssuances(ctx, ex.deltas, ex.confirmed)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}

			for aid, want := range ex.want {
				asset, err := GetAsset(ctx, aid)
				if err != nil {
					t.Fatal("unexpected error:", err)
				}
				if asset.Circulation != want {
					t.Errorf("asset %v got %v want %v", aid, asset.Circulation, want)
				}
			}
		})
	}
}

func TestUpdateAsset(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO issuer_nodes (id, project_id, key_index, keyset, label)
			VALUES ('in-id-0', 'proj-id-0', 0, '{}', 'in-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, issuance_script, label)
			VALUES ('asset-id-0', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0');
		INSERT INTO issuance_totals (asset_id) VALUES ('asset-id-0');
	`
	withContext(t, sql, func(ctx context.Context) {
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
			VALUES ('in-id-0', 'proj-id-0', 0, '{}', 'in-0');
		INSERT INTO assets (id, issuer_node_id, key_index, redeem_script, issuance_script, label)
			VALUES ('asset-id-0', 'in-id-0', 0, '\x'::bytea, '\x'::bytea, 'asset-0');
		INSERT INTO issuance_totals (asset_id) VALUES ('asset-id-0');
	`
	withContext(t, sql, func(ctx context.Context) {
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
	withContext(t, "", func(ctx context.Context) {
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

func TestAssetBalance(t *testing.T) {
	const fix = `
		INSERT INTO utxos
			(txid, index, asset_id, amount, addr_index, account_id, manager_node_id, script)
		VALUES
			('ctx-0', 0, 'asset-0', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-1', 0, 'asset-0', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-2', 0, 'asset-0', 1, 0, 'account-1', 'mnode-1', ''),
			('ctx-3', 0, 'asset-2', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-4', 0, 'asset-3', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-5', 0, 'asset-5', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-6', 0, 'asset-5', 1, 0, 'account-0', 'mnode-0', ''),
			('ctx-7', 0, 'asset-5', 1, 0, 'account-0', 'mnode-0', '');

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('ptx-0', ''), ('ptx-1', ''), ('ptx-2', ''),
			('ptx-3', ''), ('ptx-4', ''), ('ptx-5', ''),
			('ptx-6', '');

		INSERT INTO pool_outputs
			(tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, script)
		VALUES
			('ptx-0', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-1', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-2', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-3', 0, 'asset-2', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-4', 0, 'asset-4', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-5', 0, 'asset-4', 1, 0, 'account-1', 'mnode-1', ''),
			('ptx-6', 0, 'asset-5', 1, 0, 'account-1', 'mnode-1', '');

		INSERT INTO pool_inputs (tx_hash, index)
		VALUES
			('ctx-6', 0),
			('ctx-7', 0),
			('ptx-1', 0),
			('ptx-6', 0);
	`
	withContext(t, fix, func(ctx context.Context) {
		cases := []struct {
			owner     AssetOwner
			accountID string
			prev      string
			limit     int
			want      []*Balance
			wantLast  string
		}{
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "",
				limit:     9999,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 2, Total: 2},
					{AssetID: "asset-1", Confirmed: 0, Total: 2},
					{AssetID: "asset-2", Confirmed: 1, Total: 2},
					{AssetID: "asset-3", Confirmed: 1, Total: 1},
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
					{AssetID: "asset-5", Confirmed: 3, Total: 1},
				},
				wantLast: "",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 2, Total: 2},
				},
				wantLast: "asset-0",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-0",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-1", Confirmed: 0, Total: 2},
				},
				wantLast: "asset-1",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-1",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-2", Confirmed: 1, Total: 2},
				},
				wantLast: "asset-2",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-2",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-3", Confirmed: 1, Total: 1},
				},
				wantLast: "asset-3",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-3",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
				},
				wantLast: "asset-4",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-4",
				limit:     1,
				want: []*Balance{
					{AssetID: "asset-5", Confirmed: 3, Total: 1},
				},
				wantLast: "asset-5",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "",
				limit:     4,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 2, Total: 2},
					{AssetID: "asset-1", Confirmed: 0, Total: 2},
					{AssetID: "asset-2", Confirmed: 1, Total: 2},
					{AssetID: "asset-3", Confirmed: 1, Total: 1},
				},
				wantLast: "asset-3",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-3",
				limit:     4,
				want: []*Balance{
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
					{AssetID: "asset-5", Confirmed: 3, Total: 1},
				},
				wantLast: "",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-0",
				prev:      "asset-5",
				limit:     4,
				want:      nil,
				wantLast:  "",
			},
			{
				owner:     OwnerAccount,
				accountID: "account-1",
				prev:      "",
				limit:     9999,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 1, Total: 1},
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
				},
				wantLast: "",
			},

			{
				owner:     OwnerManagerNode,
				accountID: "mnode-0",
				prev:      "",
				limit:     9999,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 2, Total: 2},
					{AssetID: "asset-1", Confirmed: 0, Total: 2},
					{AssetID: "asset-2", Confirmed: 1, Total: 2},
					{AssetID: "asset-3", Confirmed: 1, Total: 1},
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
					{AssetID: "asset-5", Confirmed: 3, Total: 1},
				},
				wantLast: "",
			},
			{
				owner:     OwnerManagerNode,
				accountID: "mnode-0",
				prev:      "asset-5",
				limit:     9999,
				want:      nil,
				wantLast:  "",
			},
			{
				owner:     OwnerManagerNode,
				accountID: "mnode-1",
				prev:      "",
				limit:     9999,
				want: []*Balance{
					{AssetID: "asset-0", Confirmed: 1, Total: 1},
					{AssetID: "asset-4", Confirmed: 0, Total: 1},
				},
				wantLast: "",
			},
			{
				owner:     OwnerManagerNode,
				accountID: "mnode-1",
				prev:      "asset-4",
				limit:     9999,
				want:      nil,
				wantLast:  "",
			},
		}

		for _, c := range cases {
			got, gotLast, err := AssetBalance(ctx, &AssetBalQuery{
				Owner:   c.owner,
				OwnerID: c.accountID,
				Prev:    c.prev,
				Limit:   c.limit,
			})
			if err != nil {
				t.Errorf("AssetBalance(%s, %s, %d): unexpected error %v", c.accountID, c.prev, c.limit, err)
				continue
			}

			if !reflect.DeepEqual(got, c.want) {
				t.Fail()
				t.Logf("AssetBalance(%s, %s, %d)", c.accountID, c.prev, c.limit)

				t.Log("Got:")
				for _, b := range got {
					t.Log(b)
				}

				t.Log("Want:")
				for _, b := range c.want {
					t.Log(b)
				}
			}

			if gotLast != c.wantLast {
				t.Errorf("AssetBalance(%s, %s, %d) last = %v want %v", c.accountID, c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestAccountBalanceByAssetID(t *testing.T) {
	const fix = `
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES ('tx-0', 0, 'asset-1', 10, 0, 'account-0', 'mnode-0'),
		       ('tx-1', 1, 'asset-1', 5, 0, 'account-0', 'mnode-0'),
		       ('tx-2', 2, 'asset-2', 1, 0, 'account-0', 'mnode-0'),
		       ('tx-3', 3, 'asset-3', 2, 0, 'account-0', 'mnode-0'),
		       ('tx-4', 4, 'asset-4', 3, 0, 'account-1', 'mnode-1');
	`

	examples := []struct {
		accountID string
		assetIDs  []string
		want      []*Balance
	}{
		{
			accountID: "account-0",
			assetIDs:  []string{"asset-1", "asset-2", "asset-3", "asset-4"},
			want: []*Balance{
				{AssetID: "asset-1", Total: 15, Confirmed: 15},
				{AssetID: "asset-2", Total: 1, Confirmed: 1},
				{AssetID: "asset-3", Total: 2, Confirmed: 2},
			},
		},
		{
			accountID: "account-0",
			assetIDs:  []string{"asset-1"},
			want: []*Balance{
				{AssetID: "asset-1", Total: 15, Confirmed: 15},
			},
		},
		{
			accountID: "account-0",
			assetIDs:  []string{"asset-4"},
			want:      nil,
		},
		{
			accountID: "account-1",
			assetIDs:  []string{"asset-1", "asset-2", "asset-3", "asset-4"},
			want: []*Balance{
				{AssetID: "asset-4", Total: 3, Confirmed: 3},
			},
		},
	}

	withContext(t, fix, func(ctx context.Context) {
		for i, ex := range examples {
			t.Log("Example", i)

			got, last, err := AssetBalance(ctx, &AssetBalQuery{
				Owner:    OwnerAccount,
				OwnerID:  ex.accountID,
				AssetIDs: ex.assetIDs,
			})
			if err != nil {
				t.Fatal("unexpected error:", err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("asset IDs:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if last != "" {
				t.Errorf("got last = %q want blank", last)
			}
		}
	})
}
