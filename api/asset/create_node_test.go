package asset_test

import (
	"testing"

	"chain/api/appdb"
	. "chain/api/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestCreateManagerNode(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer pgtest.Finish(ctx)

	req := &CreateNodeReq{
		Label:        "foo",
		SigsRequired: 1,
		Keys: []*CreateNodeKeySpec{
			{Type: "node", Generate: true},
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
