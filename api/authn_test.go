package api

import (
	"testing"
	"time"

	"chain/api/asset/assettest"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
)

func TestAuthenticateToken(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	expires := time.Now().Add(-1 * time.Minute)
	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")
	tok0 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", nil)
	tok1 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", &expires)

	// Valid token
	gotUID, err := authenticateToken(ctx, tok0.ID, tok0.Secret)
	if err != nil {
		t.Errorf("correct token err = %v want nil", err)
	}

	if gotUID != uid {
		t.Errorf("correct token authenticated user id = %v want %v", gotUID, uid)
	}

	// Non-existent ID
	_, err = authenticateToken(ctx, "sample-token-id-nonexistent", "0123456789ABCDEF")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token ID err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Bad secret
	_, err = authenticateToken(ctx, tok0.ID, "bad-secret")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token secret err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Expired token
	_, err = authenticateToken(ctx, tok1.ID, tok1.Secret)
	if err != authn.ErrNotAuthenticated {
		t.Errorf("expired token err = %v want %v", err, authn.ErrNotAuthenticated)
	}
}
