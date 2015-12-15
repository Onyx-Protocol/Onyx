package appdb

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/database/pg/pgtest"
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
	err := pg.FromContext(ctx).QueryRow(ctx, q, id).Scan(&tok.secretHash, &tok.typ, &tok.userID, &expiresAt)
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
			ctx := pgtest.NewContext(t, authTokenUserFixture)
			defer pgtest.Finish(ctx)

			tok, err := CreateAuthToken(ctx, "sample-user-id-0", "sample-type-0", expiresAt)
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
				typ:        "sample-type-0",
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

func TestListAuthTokens(t *testing.T) {
	ctx := pgtest.NewContext(t, authTokenUserFixture, authTokenFixture)
	defer pgtest.Finish(ctx)

	ts, err := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	if err != nil {
		panic(err)
	}

	examples := []struct {
		userID string
		typ    string
		want   []*AuthToken
	}{
		{
			"sample-user-id-0",
			"sample-type-0",
			[]*AuthToken{
				{ID: "sample-token-id-0", CreatedAt: ts},
				{ID: "sample-token-id-1", CreatedAt: ts},
			},
		},
		{
			"sample-user-id-0",
			"sample-type-1",
			[]*AuthToken{
				{ID: "sample-token-id-2", CreatedAt: ts},
			},
		},
		{
			"sample-user-id-0",
			"nonexistent-type",
			nil,
		},
		{
			"nonexistent-user",
			"sample-type-0",
			nil,
		},
	}

	for _, ex := range examples {
		t.Log("user ID:", ex.userID, "type", ex.typ)

		got, err := ListAuthTokens(ctx, ex.userID, ex.typ)
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != len(ex.want) {
			t.Errorf("token count got=%v want=%v", len(got), len(ex.want))
			continue
		}

		for i, g := range got {
			w := ex.want[i]

			if !g.CreatedAt.Equal(w.CreatedAt) {
				t.Errorf("created at %d got=%v want=%v", i, g, w.CreatedAt)
			}

			g.CreatedAt = w.CreatedAt
			if *g != *w {
				t.Errorf("token:\ngot:  %v\nwant: %v", *g, *w)
			}
		}
	}
}

func TestDeleteAuthToken(t *testing.T) {
	ctx := pgtest.NewContext(t, authTokenUserFixture, authTokenFixture)
	defer pgtest.Finish(ctx)

	if !authTokenExists(ctx, "sample-token-id-0") {
		t.Error("initial check: token does not exist")
	}

	err := DeleteAuthToken(ctx, "sample-token-id-0")
	if err != nil {
		t.Fatal(err)
	}

	if authTokenExists(ctx, "sample-token-id-0") {
		t.Error("after delete: token still exists")
	}

	if !authTokenExists(ctx, "sample-token-id-1") {
		t.Error("after delete: other token was deleted")
	}
}

func authTokenExists(ctx context.Context, id string) bool {
	q := `SELECT 1 FROM auth_tokens WHERE id = $1`
	err := pg.FromContext(ctx).QueryRow(ctx, q, id).Scan(new(int))
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		panic(err)
	}
	return true
}
