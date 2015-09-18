package api

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
	"testing"

	"golang.org/x/net/context"
)

const (
	authTokenUserFixture = `
		INSERT INTO users (id, email, password_hash) VALUES (
			'sample-user-id-0',
			'foo@bar.com',
			'$2a$08$cHBfwMUAhhPcphRz1HgidO.gxb8WKXqUPVWfmcsuHUQoEB2RRzeSC'::bytea -- plaintext: abracadbra
		);
	`

	authTokenFixture = `
		INSERT INTO auth_tokens (id, secret_hash, type, user_id, created_at, expires_at) VALUES (
			'sample-token-id-0',
			'$2a$08$XMDacphqs44K0pzrSQxgqu3dAF.I3vn54toLboBSCKW6oSGitjSpa'::bytea, -- plaintext: 0123456789ABCDEF
			'sample-type-0',
			'sample-user-id-0',
			'2000-01-01 00:00:00+00',
			NULL
		), (
			-- expired token
			'sample-token-id-1',
			'$2a$08$XMDacphqs44K0pzrSQxgqu3dAF.I3vn54toLboBSCKW6oSGitjSpa'::bytea, -- plaintext: 0123456789ABCDEF
			'sample-type-0',
			'sample-user-id-0',
			'2000-01-01 00:00:00+00',
			'2000-01-01 00:00:00+00'
		), (
			'sample-token-id-2',
			'$2a$08$XMDacphqs44K0pzrSQxgqu3dAF.I3vn54toLboBSCKW6oSGitjSpa'::bytea, -- plaintext: 0123456789ABCDEF
			'sample-type-1',
			'sample-user-id-0',
			'2000-01-01 00:00:00+00',
			NULL
		);
	`
)

func TestAuthenticateToken(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authTokenUserFixture, authTokenFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	// Valid token
	uid, err := authenticateToken(ctx, "sample-token-id-0", "0123456789ABCDEF")
	if err != nil {
		t.Errorf("correct token err = %v want nil", err)
	}

	if uid != "sample-user-id-0" {
		t.Errorf("correct token authenticated user id = %v want sample-user-id-0", uid)
	}

	// Non-existent ID
	_, err = authenticateToken(ctx, "sample-token-id-nonexistent", "0123456789ABCDEF")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token ID err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Bad secret
	_, err = authenticateToken(ctx, "sample-token-id-0", "bad-secret")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token secret err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Expired token
	_, err = authenticateToken(ctx, "sample-token-id-1", "0123456789ABCDEF")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("expired token err = %v want %v", err, authn.ErrNotAuthenticated)
	}
}
