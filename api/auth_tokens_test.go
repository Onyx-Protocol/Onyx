package api

import (
	"testing"

	"chain/database/pg/pgtest"
	"chain/net/http/authn"
)

func TestCreateAPIToken(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	tok, err := createAPIToken(ctx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	// Verify that the token is valid
	uid, err := authenticateToken(ctx, tok.ID, tok.Secret)
	if err != nil {
		t.Errorf("authenticate token err = %v want nil", err)
	}
	if uid != "sample-user-id-0" {
		t.Errorf("authenticated user ID = %v want sample-user-id-0", uid)
	}
}
