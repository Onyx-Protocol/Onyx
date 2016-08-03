package appdb_test

// Utility functions for testing the appdb package.

import (
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
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
