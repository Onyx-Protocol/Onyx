package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
)

const testUserFixture = `
	INSERT INTO users (id, email, password_hash) VALUES (
		'sample-user-id-0',
		'foo@bar.com',
		'$2a$08$WF7tWRx/26m9Cp2kQBQEwuKxCev9S4TSzWdmtNmHSvan4UhEw0Er.'::bytea -- plaintext: abracadabra
	);
`

func TestCreateUser(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, "")
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	rec := httptest.NewRecorder()
	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBufferString(`{"email": "foo@bar.com", "password": "abracadabra"}`)),
	}

	createUser(ctx, rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %v want 200", rec.Code)
	}

	var u appdb.User
	err := readJSON(rec.Body, &u)
	if err != nil {
		t.Fatal(err)
	}

	if u.Email != "foo@bar.com" {
		t.Errorf("got email = %v want foo@bar.com", u.Email)
	}
}

func TestLogin(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	rec := httptest.NewRecorder()
	login(ctx, rec, new(http.Request))

	if rec.Code != 200 {
		t.Fatalf("status = %v want 200", rec.Code)
	}

	// Verify that the token is valid
	tok := new(appdb.AuthToken)
	err := json.NewDecoder(rec.Body).Decode(tok)
	if err != nil {
		t.Fatal(err)
	}

	uid, err := authenticateToken(ctx, tok.ID, tok.Secret)
	if err != nil {
		t.Errorf("authenticate token err = %v want nil", err)
	}

	if uid != "sample-user-id-0" {
		t.Errorf("authenticated user ID = %v want sample-user-id-0", uid)
	}
}

func TestCreateWalletBadXPub(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	const body = `{"label": "foo", "xpubs": ["badxpub"]}`
	req, _ := http.NewRequest("POST", "/v3/applications/a1/wallets", strings.NewReader(body))
	resp := httptest.NewRecorder()
	createWallet(ctx, resp, req)
	if resp.Code != 400 {
		t.Errorf("createWallet(%#q) http status = %d want 400", body, resp.Code)
	}
	want := errorInfoTab[appdb.ErrBadXPub].ChainCode
	if g := strings.TrimSpace(resp.Body.String()); !strings.Contains(g, want) {
		t.Errorf("createWallet(%#q) can't find %s in response %#q", body, want, g)
	}
}
