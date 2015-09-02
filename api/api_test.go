package api

import (
	"bytes"
	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

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
