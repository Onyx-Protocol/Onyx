package accesstoken

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()
	cs := &CredentialStore{DB: pgtest.NewTx(t)}

	cases := []struct {
		id, net string
		want    error
	}{
		{"a", "client", nil},
		{"b", "network", nil},
		{"", "client", ErrBadID},
		{"bad:id", "client", ErrBadID},
		{"a", "network", ErrDuplicateID}, // this aborts the transaction, so no tests can follow
	}

	for _, c := range cases {
		_, err := cs.Create(ctx, c.id, c.net)
		if errors.Root(err) != c.want {
			t.Errorf("Create(%s, %s) error = %s want %s", c.id, c.net, err, c.want)
		}
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	cs := &CredentialStore{DB: pgtest.NewTx(t)}
	a := mustCreateToken(t, ctx, cs, "a", "client")
	b := mustCreateToken(t, ctx, cs, "b", "network")
	c := mustCreateToken(t, ctx, cs, "c", "client")
	for _, token := range []*Token{a, b, c} {
		token.Token = ""
	}

	cases := []struct {
		typ      string
		after    string
		limit    int
		want     []*Token
		wantNext string
	}{{
		limit:    100,
		want:     []*Token{c, b, a},
		wantNext: a.sortID,
	}, {
		limit:    1,
		want:     []*Token{c},
		wantNext: c.sortID,
	}, {
		after:    c.sortID,
		limit:    1,
		want:     []*Token{b},
		wantNext: b.sortID,
	}, {
		typ:      "client",
		limit:    100,
		want:     []*Token{c, a},
		wantNext: a.sortID,
	}, {
		typ:      "client",
		after:    c.sortID,
		limit:    1,
		want:     []*Token{a},
		wantNext: a.sortID,
	}, {
		typ:      "network",
		limit:    100,
		want:     []*Token{b},
		wantNext: b.sortID,
	}}

	for _, c := range cases {
		got, gotNext, err := cs.List(ctx, c.typ, c.after, c.limit)

		if err != nil {
			t.Errorf("List(%s, %d) errored: %s", c.after, c.limit, err)
			continue
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("List(%s, %d) = %+v want %+v", c.after, c.limit, spew.Sdump(got), spew.Sdump(c.want))
		}

		if gotNext != c.wantNext {
			t.Errorf("List(%s, %d) next = %q want %q", c.after, c.limit, gotNext, c.wantNext)
		}
	}
}

func TestCheck(t *testing.T) {
	ctx := context.Background()
	cs := &CredentialStore{DB: pgtest.NewTx(t)}

	token := mustCreateToken(t, ctx, cs, "x", "client")

	tokenParts := strings.Split(token.Token, ":")
	tokenID := tokenParts[0]
	tokenSecret, err := hex.DecodeString(tokenParts[1])
	if err != nil {
		t.Fatal("bad token secret")
	}

	valid, err := cs.Check(ctx, tokenID, tokenSecret)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected token and secret to be valid")
	}

	valid, err = cs.Check(ctx, "x", []byte("badsecret"))
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("expected bad secret to not be valid")
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	cs := &CredentialStore{DB: pgtest.NewTx(t)}

	token := mustCreateToken(t, ctx, cs, "x", "client")
	err := cs.Delete(ctx, token.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func mustCreateToken(t *testing.T, ctx context.Context, cs *CredentialStore, id, typ string) *Token {
	token, err := cs.Create(ctx, id, typ)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
