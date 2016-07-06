package appdb_test

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/lib/pq"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
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
	err := pg.QueryRow(ctx, q, id).Scan(&tok.secretHash, &tok.typ, &tok.userID, &expiresAt)
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
			ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

			uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra", "admin")

			tok, err := CreateAuthToken(ctx, uid, "sample-type-0", expiresAt)
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
				userID:     uid,
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ts, err := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra", "admin")
	tok0 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", nil)
	tok1 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", &ts)
	tok2 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-1", nil)

	examples := []struct {
		userID string
		typ    string
		want   []*AuthToken
	}{
		{
			uid,
			"sample-type-0",
			[]*AuthToken{
				{ID: tok0.ID, CreatedAt: tok0.CreatedAt},
				{ID: tok1.ID, CreatedAt: tok1.CreatedAt},
			},
		},
		{
			uid,
			"sample-type-1",
			[]*AuthToken{
				{ID: tok2.ID, CreatedAt: tok2.CreatedAt},
			},
		},
		{
			uid,
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ts, err := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra", "admin")
	tok0 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", nil)
	tok1 := assettest.CreateAuthTokenFixture(ctx, t, uid, "sample-type-0", &ts)

	if _, _, _, err := GetAuthToken(ctx, tok0.ID); err != nil {
		t.Error("initial check: token does not exist")
	}

	err = DeleteAuthToken(ctx, tok0.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := GetAuthToken(ctx, tok0.ID); err == nil {
		t.Error("after delete: token still exists")
	}

	if _, _, _, err := GetAuthToken(ctx, tok1.ID); err != nil {
		t.Error("after delete: other token was deleted")
	}
}
