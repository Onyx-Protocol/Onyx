package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

func TestInsertManagerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		_ = newTestManagerNode(t, ctx, nil, "foo")
	})
}

func TestGetManagerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		proj := newTestProject(t, ctx, "foo", nil)
		mn, err := InsertManagerNode(ctx, proj.ID, "manager-node-0", []*hdkey.XKey{dummyXPub}, []*hdkey.XKey{dummyXPrv}, 0, 1)

		if err != nil {
			t.Fatalf("unexpected error on InsertManagerNode: %v", err)
		}
		examples := []struct {
			id      string
			want    *ManagerNode
			wantErr error
		}{
			{
				mn.ID,
				&ManagerNode{
					ID:    mn.ID,
					Label: "manager-node-0",
					Keys: []*NodeKey{
						{
							Type: "node",
							XPub: dummyXPub,
							XPrv: dummyXPrv,
						},
					},
					SigsReqd: 1,
				},
				nil,
			},
			{
				"nonexistent",
				nil,
				pg.ErrUserInputNotFound,
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.id)

			got, gotErr := GetManagerNode(ctx, ex.id)

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("managerNode:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("get managerNode error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
			}
		}
	})
}

func TestAccountsWithAsset(t *testing.T) {
	const fix = `
	INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0'),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1');

		INSERT INTO accounts (id, manager_node_id, key_index, label, archived) VALUES
			('account-0', 'manager-node-id-0', 0, 'account-0', false),
			('account-1', 'manager-node-id-0', 1, 'account-1', false);

		INSERT INTO utxos (tx_hash, index, asset_id, amount, addr_index, manager_node_id, account_id, confirmed, block_hash, block_height)
		VALUES ('ctx-0', 0, 'asset-1', 5, 0, 'mnode-0', 'account-0', TRUE, 'bh1', 1),
		       ('ctx-1', 0, 'asset-1', 5, 0, 'mnode-0', 'account-0', TRUE, 'bh1', 1),
		       ('ctx-2', 0, 'asset-1', 5, 0, 'mnode-0', 'account-1', TRUE, 'bh1', 1),
		       ('ctx-3', 0, 'asset-2', 5, 0, 'mnode-0', 'account-1', TRUE, 'bh1', 1),
		       ('ctx-4', 0, 'asset-1', 5, 0, 'mnode-1', 'account-0', TRUE, 'bh1', 1),
		       ('ctx-5', 0, 'asset-1', 5, 0, 'mnode-1', 'account-2', TRUE, 'bh1', 1);

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('ptx-0', ''), ('ptx-1', '');

		INSERT INTO utxos
			(tx_hash, pool_tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, script, confirmed)
		VALUES
			('ptx-0', 'ptx-0', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', '', FALSE),
			('ptx-1', 'ptx-1', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', '', FALSE);

		INSERT INTO pool_inputs (tx_hash, index)
		VALUES ('ptx-1', 0), ('ctx-3', 0);

	`
	withContext(t, fix, func(ctx context.Context) {
		cases := []struct {
			assetID  string
			prev     string
			limit    int
			want     []*AccountBalanceItem
			wantLast string
		}{{
			assetID: "asset-1",
			prev:    "",
			limit:   50,
			want: []*AccountBalanceItem{
				{"account-0", 10, 11},
				{"account-1", 5, 5},
			},
			wantLast: "account-1",
		}, {
			assetID: "asset-1",
			prev:    "account-0",
			limit:   50,
			want: []*AccountBalanceItem{
				{"account-1", 5, 5},
			},
			wantLast: "account-1",
		}, {
			assetID: "asset-1",
			prev:    "",
			limit:   1,
			want: []*AccountBalanceItem{
				{"account-0", 10, 11},
			},
			wantLast: "account-0",
		}, {
			assetID:  "asset-1",
			prev:     "account-1",
			limit:    50,
			want:     nil,
			wantLast: "",
		}, {
			assetID: "asset-2",
			prev:    "",
			limit:   50,
			want: []*AccountBalanceItem{
				{"account-1", 5, 0},
			},
			wantLast: "account-1",
		}}
		for _, c := range cases {
			got, gotLast, err := AccountsWithAsset(ctx, "mnode-0", c.assetID, c.prev, c.limit)
			if err != nil {
				t.Errorf("AccountsWithAsset(%q, %d) unexpected error = %q", c.prev, c.limit, err)
				continue
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("AccountsWithAsset(%q, %d) = %+v want %+v", c.prev, c.limit, got, c.want)
			}
			if gotLast != c.wantLast {
				t.Errorf("AccountsWithAsset(%q, %d) last = %q want %q", c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestListManagerNodes(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO manager_nodes (id, project_id, key_index, label, created_at) VALUES
			-- insert in reverse chronological order, to ensure that ListManagerNodes
			-- is performing a sort.
			('manager-node-id-2', 'proj-id-1', 2, 'manager-node-2', now()),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1', now()),
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0', now());
	`
	withContext(t, sql, func(ctx context.Context) {
		examples := []struct {
			projID string
			want   []*ManagerNode
		}{
			{
				"proj-id-0",
				[]*ManagerNode{
					{ID: "manager-node-id-0", Label: "manager-node-0"},
					{ID: "manager-node-id-1", Label: "manager-node-1"},
				},
			},
			{
				"proj-id-1",
				[]*ManagerNode{
					{ID: "manager-node-id-2", Label: "manager-node-2"},
				},
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.projID)

			got, err := ListManagerNodes(ctx, ex.projID)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("managerNodes:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestUpdateManagerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		newLabel := "bar"

		err := UpdateManagerNode(ctx, managerNode.ID, &newLabel)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}

		managerNode, err = GetManagerNode(ctx, managerNode.ID)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if managerNode.Label != newLabel {
			t.Errorf("Expected %s, got %s", newLabel, managerNode.Label)
		}
	})
}

// Test that calling UpdateManagerNode with no new label is a no-op.
func TestUpdateManagerNodeNoUpdate(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		err := UpdateManagerNode(ctx, managerNode.ID, nil)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		managerNode, err = GetManagerNode(ctx, managerNode.ID)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if managerNode.Label != "foo" {
			t.Errorf("Expected foo, got %s", managerNode.Label)
		}
	})
}

func TestArchiveManagerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		account := newTestAccount(t, ctx, managerNode, "bar")
		err := ArchiveManagerNode(ctx, managerNode.ID)
		if err != nil {
			t.Errorf("could not archive manager node with id %s: %v", managerNode.ID, err)
		}

		var archived bool
		checkQ := `SELECT archived FROM manager_nodes WHERE id = $1`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ, managerNode.ID).Scan(&archived)

		if !archived {
			t.Errorf("expected manager node %s to be archived", managerNode.ID)
		}

		checkAccountQ := `SELECT archived FROM accounts WHERE id = $1`
		err = pg.FromContext(ctx).QueryRow(ctx, checkAccountQ, account.ID).Scan(&archived)
		if !archived {
			t.Errorf("expected child account %s to be archived", account.ID)
		}

	})
}
