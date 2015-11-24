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
	withContext(t, "", func(t *testing.T, ctx context.Context) {
		_ = newTestManagerNode(t, ctx, nil, "foo")
	})
}

func TestGetManagerNode(t *testing.T) {
	withContext(t, "", func(t *testing.T, ctx context.Context) {
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

func TestManagerNodeBalance(t *testing.T) {
	const sql = `
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES ('t0', 0, 'a1', 10, 0, 'b0', 'mn1'),
		       ('t1', 1, 'a1', 5, 0, 'b0', 'mn1'),
		       ('t2', 2, 'a2', 20, 0, 'b1', 'mn1');
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		cases := []struct {
			mnID     string
			prev     string
			limit    int
			want     []*Balance
			wantLast string
		}{{
			mnID:     "mn1",
			limit:    5,
			want:     []*Balance{{"a1", 15, 15}, {"a2", 20, 20}},
			wantLast: "a2",
		}, {
			mnID:     "mn1",
			prev:     "a1",
			limit:    5,
			want:     []*Balance{{"a2", 20, 20}},
			wantLast: "a2",
		}, {
			mnID:     "mn1",
			prev:     "a2",
			limit:    5,
			want:     nil,
			wantLast: "",
		}, {
			mnID:     "mn1",
			limit:    1,
			want:     []*Balance{{"a1", 15, 15}},
			wantLast: "a1",
		}, {
			mnID:     "nonexistent",
			limit:    5,
			want:     nil,
			wantLast: "",
		}}

		for _, c := range cases {
			got, gotLast, err := ManagerNodeBalance(ctx, c.mnID, c.prev, c.limit)
			if err != nil {
				t.Errorf("ManagerNodeBalance(%s, %s, %d): unexpected error %v", c.mnID, c.prev, c.limit, err)
				continue
			}

			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ManagerNodeBalance(%s, %s, %d) = %v want %v", c.mnID, c.prev, c.limit, got, c.want)
			}

			if gotLast != c.wantLast {
				t.Errorf("ManagerNodeBalance(%s, %s, %d) = %v want %v", c.mnID, c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestAccountsWithAsset(t *testing.T) {
	const fix = `
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, manager_node_id, account_id)
		VALUES ('t0', 0, 'a0', 5, 0, 'mn0', 'acc0'),
		       ('t1', 0, 'a0', 5, 0, 'mn0', 'acc0'),
		       ('t2', 0, 'a0', 5, 0, 'mn0', 'acc1'),
		       ('t3', 0, 'a1', 5, 0, 'mn0', 'acc1'),
		       ('t4', 0, 'a0', 5, 0, 'mn1', 'acc0');
	`
	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		cases := []struct {
			prev     string
			limit    int
			want     []*AccountBalanceItem
			wantLast string
		}{{
			prev:  "",
			limit: 50,
			want: []*AccountBalanceItem{
				{"acc0", 10, 10},
				{"acc1", 5, 5},
			},
			wantLast: "acc1",
		}, {
			prev:  "acc0",
			limit: 50,
			want: []*AccountBalanceItem{
				{"acc1", 5, 5},
			},
			wantLast: "acc1",
		}, {
			prev:  "",
			limit: 1,
			want: []*AccountBalanceItem{
				{"acc0", 10, 10},
			},
			wantLast: "acc0",
		}, {
			prev:     "acc1",
			limit:    50,
			want:     nil,
			wantLast: "",
		}}
		for _, c := range cases {
			got, gotLast, err := AccountsWithAsset(ctx, "mn0", "a0", c.prev, c.limit)
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
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0', now()),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1', now() - '1 day'::interval),

			('manager-node-id-2', 'proj-id-1', 2, 'manager-node-2', now());
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			projID string
			want   []*ManagerNode
		}{
			{
				"proj-id-0",
				[]*ManagerNode{
					{ID: "manager-node-id-1", Blockchain: "sandbox", Label: "manager-node-1"},
					{ID: "manager-node-id-0", Blockchain: "sandbox", Label: "manager-node-0"},
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
	withContext(t, "", func(t *testing.T, ctx context.Context) {
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
	withContext(t, "", func(t *testing.T, ctx context.Context) {
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
	withContext(t, "", func(t *testing.T, ctx context.Context) {
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
	withContext(t, "", func(t *testing.T, ctx context.Context) {
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
