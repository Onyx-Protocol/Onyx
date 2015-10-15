package api

import (
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
)

const testUserFixture = `
	INSERT INTO users (id, email, password_hash) VALUES (
		'sample-user-id-0',
		'foo@bar.com',
		'$2a$08$WF7tWRx/26m9Cp2kQBQEwuKxCev9S4TSzWdmtNmHSvan4UhEw0Er.'::bytea -- plaintext: abracadabra
	);
`

func TestMux(t *testing.T) {
	// Handler calls httpjson.HandleFunc, which panics
	// if the function signature is not of the right form.
	// So call Handler here and rescue any panic
	// to check for this case.
	defer func() {
		if err := recover(); err != nil {
			t.Fatal("unexpected panic:", err)
		}
	}()
	Handler("")
}

func TestLogin(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	tok, err := login(ctx)
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

func TestCreateWalletBadXPub(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture, `
		INSERT INTO projects(id, name) VALUES ('a1', 'x');
		INSERT INTO members (project_id, user_id, role)
			VALUES ('a1', 'sample-user-id-0', 'admin');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	req := &asset.CreateNodeReq{
		Label: "foo",
		XPubs: []string{"badxpub"},
	}

	_, err := createWallet(ctx, "a1", req)
	if got := errors.Root(err); got != asset.ErrBadXPub {
		t.Fatalf("err = %v want %v", got, asset.ErrBadXPub)
	}
}
