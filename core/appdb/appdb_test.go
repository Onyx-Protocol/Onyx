package appdb_test

// Utility functions for testing the appdb package.

import (
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
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
