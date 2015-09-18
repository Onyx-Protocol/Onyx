package api

import (
	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/net/http/authn"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

func TestCreateAPIToken(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, testUserFixture)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	rec := httptest.NewRecorder()
	createAPIToken(ctx, rec, new(http.Request))

	if rec.Code != 200 {
		t.Fatalf("response code = %v want 200", rec.Code)
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
