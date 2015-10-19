package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

func TestInsertManagerNode(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, sampleProjectFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	managerNode, err := InsertManagerNode(ctx, "proj-id-0", "foo", []*hdkey.XKey{dummyXPub}, nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if managerNode.ID == "" {
		t.Errorf("got empty managerNode id")
	}
}

func TestGetManagerNode(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	examples := []struct {
		id      string
		want    *ManagerNode
		wantErr error
	}{
		{
			"manager-node-id-0",
			&ManagerNode{ID: "manager-node-id-0", Label: "manager-node-0", Blockchain: "sandbox"},
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
}

func TestManagerNodeBalance(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES ('t0', 0, 'a1', 10, 0, 'b0', 'mn1'),
		       ('t1', 1, 'a1', 5, 0, 'b0', 'mn1'),
		       ('t2', 2, 'a2', 20, 0, 'b1', 'mn1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
}

func TestListManagerNodes(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0'),
			('proj-id-1', 'proj-1');

		INSERT INTO manager_nodes (id, project_id, key_index, label, created_at) VALUES
			-- insert in reverse chronological order, to ensure that ListManagerNodes
			-- is performing a sort.
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0', now()),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1', now() - '1 day'::interval),

			('manager-node-id-2', 'proj-id-1', 2, 'manager-node-2', now());
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

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
}
