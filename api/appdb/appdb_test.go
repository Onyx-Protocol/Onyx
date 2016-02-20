package appdb_test

// Utility functions for testing the appdb package.

import (
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain-sandbox/hdkey"
)

// Establish a context object with a new db transaction in which to
// run the given callback function.
func withContext(tb testing.TB, sql string, fn func(context.Context)) {
	var ctx context.Context
	if sql == "" {
		ctx = pgtest.NewContext(tb)
	} else {
		ctx = pgtest.NewContext(tb, sql)
	}
	defer pgtest.Finish(ctx)
	fn(ctx)
}

func newTestUser(t *testing.T, ctx context.Context, email, password string) *User {
	ensureInTransaction(ctx)
	if email == "" {
		email = "foo@bar.com"
	}
	if password == "" {
		password = "a valid password"
	}
	user, err := CreateUser(ctx, email, password)
	if err != nil {
		t.Fatalf("trouble setting up user in newTestUser: %v", err)
	}
	return user
}

func newTestProject(t *testing.T, ctx context.Context, name string, user *User) *Project {
	ensureInTransaction(ctx)
	if user == nil {
		user = newTestUser(t, ctx, "", "")
	}
	project, err := CreateProject(ctx, name, user.ID)
	if err != nil {
		t.Fatalf("trouble setting up project in newTestProject: %v", err)
	}
	return project
}

func newTestIssuerNode(t *testing.T, ctx context.Context, project *Project, label string) *IssuerNode {
	ensureInTransaction(ctx)
	if project == nil {
		project = newTestProject(t, ctx, "project-1", nil)
	}
	issuerNode, err := InsertIssuerNode(ctx, project.ID, label, []*hdkey.XKey{dummyXPub}, nil, 1, nil)
	if err != nil {
		t.Fatalf("trouble setting up issuer node in newTestIssuerNode: %v", err)
	}
	if issuerNode.ID == "" {
		t.Fatal("got empty issuer node id in newTestIssuerNode")
	}
	return issuerNode
}

func newTestManagerNode(t *testing.T, ctx context.Context, project *Project, label string) *ManagerNode {
	ensureInTransaction(ctx)
	if project == nil {
		project = newTestProject(t, ctx, "project-1", nil)
	}
	managerNode, err := InsertManagerNode(ctx, project.ID, label, []*hdkey.XKey{dummyXPub}, nil, 0, 1, nil)
	if err != nil {
		t.Fatalf("could not create manager node in newTestManagerNode: %v", err)
	}
	if managerNode.ID == "" {
		t.Fatal("got empty manager node id in newTestManagerNode")
	}
	return managerNode
}

func newTestVarKeyManagerNode(t *testing.T, ctx context.Context, project *Project, label string, varKeys, sigsReq int) *ManagerNode {
	ensureInTransaction(ctx)
	if project == nil {
		project = newTestProject(t, ctx, "project-1", nil)
	}
	managerNode, err := InsertManagerNode(ctx, project.ID, label, []*hdkey.XKey{dummyXPub}, nil, varKeys, sigsReq, nil)
	if err != nil {
		t.Fatalf("could not create manager node in newTestVarKeyManagerNode: %v", err)
	}
	if managerNode.ID == "" {
		t.Fatal("got empty manager node id in newTestVarKeyManagerNode")
	}
	return managerNode
}

func newTestAccount(t *testing.T, ctx context.Context, managerNode *ManagerNode, label string) *Account {
	ensureInTransaction(ctx)
	if managerNode == nil {
		managerNode = newTestManagerNode(t, ctx, nil, "manager-node-1")
	}
	account, err := CreateAccount(ctx, managerNode.ID, label, nil, nil)
	if err != nil {
		t.Fatalf("could not create account in newTestAccount: %v", err)
	}
	return account
}

func newTestAsset(t *testing.T, ctx context.Context, issuerNode *IssuerNode) *Asset {
	ensureInTransaction(ctx)
	if issuerNode == nil {
		issuerNode = newTestIssuerNode(t, ctx, nil, "issuer-node-1")
	}
	asset, _, err := NextAsset(ctx, issuerNode.ID)
	if err != nil {
		t.Fatalf("trouble setting up asset in newTestAsset: %v", err)
	}
	asset, err = InsertAsset(ctx, asset)
	if err != nil {
		t.Fatalf("trouble setting up asset in newTestAsset: %v", err)
	}
	return asset
}

// Panics if not in a transaction
func ensureInTransaction(ctx context.Context) {
	_ = pg.FromContext(ctx).(pg.Tx)
}
