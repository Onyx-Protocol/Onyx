package appdb

// Utility functions for testing the appdb package.

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

// Establish a context object with a new db transaction in which to
// run the given callback function.
func withContext(t *testing.T, sql string, fn func(*testing.T, context.Context)) {
	var dbtx pg.Tx
	if sql == "" {
		dbtx = pgtest.TxWithSQL(t)
	} else {
		dbtx = pgtest.TxWithSQL(t, sql)
	}
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	fn(t, ctx)
}

func newTestIssuerNode(t *testing.T, ctx context.Context, inodeID, label string) *AssetGroup {
	ensureInTransaction(ctx)
	issuerNode, err := InsertAssetGroup(ctx, inodeID, label, []*hdkey.XKey{dummyXPub}, nil)
	if err != nil {
		t.Fatalf("trouble setting up issuer node in withIssuerNode: %v", err)
	}
	if issuerNode.ID == "" {
		t.Fatal("got empty issuer node id in withIssuerNode")
	}
	return issuerNode
}

func newTestManagerNode(t *testing.T, ctx context.Context, mnodeID, label string) *ManagerNode {
	ensureInTransaction(ctx)
	managerNode, err := InsertManagerNode(ctx, mnodeID, label, []*hdkey.XKey{dummyXPub}, nil)
	if err != nil {
		t.Fatalf("could not create manager node in withManagerNode: %v", err)
	}
	if managerNode.ID == "" {
		t.Fatal("got empty manager node id in withManagerNode")
	}
	return managerNode
}

// Panics if not in a transaction
func ensureInTransaction(ctx context.Context) {
	_ = pg.FromContext(ctx).(pg.Tx)
}
