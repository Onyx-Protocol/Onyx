package appdb_test

// Utility functions for testing the appdb package.

import (
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/testutil"
)

func newTestUser(t *testing.T, ctx context.Context, email, password, role string) *User {
	if email == "" {
		email = "foo@bar.com"
	}
	if password == "" {
		password = "a valid password"
	}
	if role == "" {
		role = "developer"
	}
	user, err := CreateUser(ctx, email, password, role)
	if err != nil {
		t.Fatalf("trouble setting up user in newTestUser: %v", err)
	}
	return user
}

func newTestProject(t *testing.T, ctx context.Context, name string) *Project {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	defer dbtx.Rollback(ctx)

	project, err := CreateProject(ctx, name)
	if err != nil {
		t.Fatalf("trouble setting up project in newTestProject: %v", err)
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return project
}

func newTestIssuerNode(t *testing.T, ctx context.Context, project *Project, label string) *IssuerNode {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	defer dbtx.Rollback(ctx)

	if project == nil {
		project = newTestProject(t, ctx, "project-1")
	}
	issuerNode, err := InsertIssuerNode(ctx, project.ID, label, []*hd25519.XPub{dummyXPub}, nil, 1, nil)
	if err != nil {
		t.Fatalf("trouble setting up issuer node in newTestIssuerNode: %v", err)
	}
	if issuerNode.ID == "" {
		t.Fatal("got empty issuer node id in newTestIssuerNode")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return issuerNode
}

func newTestManagerNode(t *testing.T, ctx context.Context, project *Project, label string) *ManagerNode {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	defer dbtx.Rollback(ctx)

	if project == nil {
		project = newTestProject(t, ctx, "project-1")
	}
	managerNode, err := InsertManagerNode(ctx, project.ID, label, []*hd25519.XPub{dummyXPub}, nil, 0, 1, nil)
	if err != nil {
		t.Fatalf("could not create manager node in newTestManagerNode: %v", err)
	}
	if managerNode.ID == "" {
		t.Fatal("got empty manager node id in newTestManagerNode")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return managerNode
}

func newTestVarKeyManagerNode(t *testing.T, ctx context.Context, project *Project, label string, varKeys, sigsReq int) *ManagerNode {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	defer dbtx.Rollback(ctx)

	if project == nil {
		project = newTestProject(t, ctx, "project-1")
	}
	managerNode, err := InsertManagerNode(ctx, project.ID, label, []*hd25519.XPub{dummyXPub}, nil, varKeys, sigsReq, nil)
	if err != nil {
		t.Fatalf("could not create manager node in newTestVarKeyManagerNode: %v", err)
	}
	if managerNode.ID == "" {
		t.Fatal("got empty manager node id in newTestVarKeyManagerNode")
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return managerNode
}

func newTestAccount(t *testing.T, ctx context.Context, managerNode *ManagerNode, label string) *Account {
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
