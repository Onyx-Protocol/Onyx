package asset

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestCreateManagerNode(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	node, err := CreateNode(ctx, ManagerNode, "proj-id-0", &CreateNodeReq{Label: "foo", GenerateKey: true})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	managerNode, ok := node.(*appdb.ManagerNode)
	if !ok {
		t.Fatal("expected ManagerNode struct")
	}
	if managerNode.ID == "" {
		t.Errorf("got empty managerNode id")
	}
	var valid bool
	const checkQ = `
		SELECT SUBSTR(generated_keys[1], 1, 4)='xprv' FROM manager_nodes LIMIT 1
	`
	err = dbtx.QueryRow(checkQ).Scan(&valid)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Errorf("private key not stored")
	}
}

func TestNewKey(t *testing.T) {
	pub, priv, err := newKey()
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	validPub, err := priv.Neuter()
	if err != nil {
		t.Fatal(err)
	}

	if validPub.String() != pub.String() {
		t.Fatal("incorrect private/public key pair")
	}
}
