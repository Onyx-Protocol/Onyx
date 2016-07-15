package asset_test

import (
	"testing"

	"golang.org/x/net/context"

	"chain/core/appdb"
	. "chain/core/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestCreateManagerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, ctx, err := pg.Begin(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	pgtest.Exec(ctx, t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)

	req := &CreateNodeReq{
		Label:        "foo",
		SigsRequired: 1,
		Keys: []*CreateNodeKeySpec{
			{Type: "service", Generate: true},
		},
	}
	node, err := CreateNode(ctx, ManagerNode, "proj-id-0", req)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	mnode, ok := node.(*appdb.ManagerNode)
	if !ok {
		t.Fatal("expected CreateNode return value to be a manager node")
	}
	if mnode.ID == "" {
		t.Errorf("got empty managerNode id")
	}

	var valid bool
	const checkQ = `
		SELECT SUBSTR(generated_keys[1], 1, 4)='xprv' FROM manager_nodes LIMIT 1
	`
	err = pg.QueryRow(ctx, checkQ).Scan(&valid)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Errorf("private key not stored")
	}
}
