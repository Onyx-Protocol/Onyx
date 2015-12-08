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
		mn, err := InsertManagerNode(ctx, proj.ID, "manager-node-0", []*hdkey.XKey{dummyXPub}, []*hdkey.XKey{dummyXPrv})
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
					ID:          mn.ID,
					Label:       "manager-node-0",
					Blockchain:  "sandbox",
					Keys:        []*hdkey.XKey{dummyXPub},
					PrivateKeys: []*hdkey.XKey{dummyXPrv},
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
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, manager_node_id, account_id)
		VALUES ('ctx-0', 0, 'asset-1', 5, 0, 'mnode-0', 'account-0'),
		       ('ctx-1', 0, 'asset-1', 5, 0, 'mnode-0', 'account-0'),
		       ('ctx-2', 0, 'asset-1', 5, 0, 'mnode-0', 'account-1'),
		       ('ctx-3', 0, 'asset-2', 5, 0, 'mnode-0', 'account-1'),
		       ('ctx-4', 0, 'asset-1', 5, 0, 'mnode-1', 'account-0');

		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('ptx-0', ''), ('ptx-1', '');

		INSERT INTO pool_outputs
			(tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, script)
		VALUES
			('ptx-0', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', ''),
			('ptx-1', 0, 'asset-1', 1, 0, 'account-0', 'mnode-0', '');

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
					{ID: "manager-node-id-0", Blockchain: "sandbox", Label: "manager-node-0"},
					{ID: "manager-node-id-1", Blockchain: "sandbox", Label: "manager-node-1"},
				},
			},
			{
				"proj-id-1",
				[]*ManagerNode{
					{ID: "manager-node-id-2", Blockchain: "sandbox", Label: "manager-node-2"},
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

func TestDeleteManagerNode(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")

		_, err := GetManagerNode(ctx, managerNode.ID)
		if err != nil {
			t.Errorf("could not get test manager node with id %s: %v", managerNode.ID, err)
		}

		err = DeleteManagerNode(ctx, managerNode.ID)
		if err != nil {
			t.Errorf("could not delete manager node with id %s: %v", managerNode.ID, err)
		}

		_, err = GetManagerNode(ctx, managerNode.ID)
		if err == nil { // sic
			t.Errorf("expected manager node %s would be deleted, but it wasn't", managerNode.ID)
		} else {
			rootErr := errors.Root(err)
			if rootErr != pg.ErrUserInputNotFound {
				t.Errorf("unexpected error when trying to get deleted manager node %s: %v", managerNode.ID, err)
			}
		}
	})
}

// Test that the existence of an account connected to a manager node
// prevents deletion of the node.
func TestDeleteManagerNodeBlocked(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, nil, "foo")
		_ = newTestAccount(t, ctx, managerNode, "bar")
		err := DeleteManagerNode(ctx, managerNode.ID)
		if err == nil { // sic
			t.Errorf("expected to be unable to delete manager node %s, but was able to", managerNode.ID)
		} else {
			rootErr := errors.Root(err)
			if rootErr != ErrCannotDelete {
				t.Errorf("unexpected error trying to delete undeletable manager node %s: %v", managerNode.ID, err)
			}
		}
	})
}
