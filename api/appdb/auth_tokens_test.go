package appdb

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
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
		INSERT INTO auth_tokens (id, secret_hash, type, user_id) VALUES (
			'sample-token-id-0',
			'$2a$08$XMDacphqs44K0pzrSQxgqu3dAF.I3vn54toLboBSCKW6oSGitjSpa'::bytea, -- plaintext: 0123456789ABCDEF
			'type-does-not-matter',
			'sample-user-id-0'
		);

		-- expired token
		INSERT INTO auth_tokens (id, secret_hash, type, user_id, expires_at) VALUES (
			'sample-token-id-1',
			'$2a$08$XMDacphqs44K0pzrSQxgqu3dAF.I3vn54toLboBSCKW6oSGitjSpa'::bytea, -- plaintext: 0123456789ABCDEF
			'type-does-not-matter',
			'sample-user-id-0',
			'2000-01-01 00:00:00+00'
		);
	`
)

type testAuthToken struct {
	id         string
	secretHash string
	typ        string
	userID     string
	expiresAt  *time.Time
}

func testGetAuthToken(ctx context.Context, id string) (*testAuthToken, error) {
	var (
		q         = `SELECT secret_hash, type, user_id, expires_at FROM auth_tokens WHERE id = $1`
		expiresAt pq.NullTime
		tok       = &testAuthToken{id: id}
	)
	err := pg.FromContext(ctx).QueryRow(q, id).Scan(&tok.secretHash, &tok.typ, &tok.userID, &expiresAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		tok.expiresAt = &t
	}
	return tok, nil
}

func TestCreateAuthToken(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	examples := []*time.Time{nil, &now}

	for _, expiresAt := range examples {
		t.Log("expiresAt:", expiresAt)

		func() {
			dbtx := pgtest.TxWithSQL(t, authTokenUserFixture)
			defer dbtx.Rollback()
			ctx := pg.NewContext(context.Background(), dbtx)

			tok, err := CreateAuthToken(ctx, "sample-user-id-0", "sample-type", expiresAt)
			if err != nil {
				t.Fatal(err)
			}

			if tok.ID == "" {
				t.Fatal("token ID is blank")
			}

			if tok.Secret == "" {
				t.Fatal("token secret is blank")
			}

			got, err := testGetAuthToken(ctx, tok.ID)
			if err != nil {
				t.Fatal(err)
			}

			want := testAuthToken{
				id:         tok.ID,
				secretHash: got.secretHash, // doesn't matter as long as it's not the secret
				typ:        "sample-type",
				userID:     "sample-user-id-0",
				expiresAt:  expiresAt,
			}

			if !reflect.DeepEqual(*got, want) {
				t.Errorf("persisted token fields:\ngot:  %v\nwant: %v", *got, want)
			}

			if got.secretHash == tok.Secret {
				t.Errorf("secret and secret hash are equal: %v", tok.Secret)
			}
		}()
	}
}

func TestAuthenticateToken(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, authTokenUserFixture, authTokenFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	// Valid token
	uid, err := AuthenticateToken(ctx, "sample-token-id-0", "0123456789ABCDEF")
	if err != nil {
		t.Errorf("correct token err = %v want nil", err)
	}

	if uid != "sample-user-id-0" {
		t.Errorf("correct token authenticated user id = %v want sample-user-id-0", uid)
	}

	// Non-existent ID
	_, err = AuthenticateToken(ctx, "sample-token-id-nonexistent", "0123456789ABCDEF")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token ID err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Bad secret
	_, err = AuthenticateToken(ctx, "sample-token-id-0", "bad-secret")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("bad token secret err = %v want %v", err, authn.ErrNotAuthenticated)
	}

	// Expired token
	_, err = AuthenticateToken(ctx, "sample-token-id-1", "0123456789ABCDEF")
	if err != authn.ErrNotAuthenticated {
		t.Errorf("expired token err = %v want %v", err, authn.ErrNotAuthenticated)
	}
}
