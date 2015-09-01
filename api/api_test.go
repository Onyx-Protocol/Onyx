package api

import (
	"bytes"
	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"encoding/json"
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
	err := json.NewDecoder(rec.Body).Decode(&u)
	if err != nil {
		t.Fatal(err)
	}

	if u.Email != "foo@bar.com" {
		t.Errorf("got email = %v want foo@bar.com", u.Email)
	}
}

func TestCreateUserValidation(t *testing.T) {
	examples := []string{
		"not json",

		// missing password
		`{"email": "foo@bar.com"}`,
		// password too short
		`{"email": "foo@bar.com", "password": "12345"}`,
		// password too long
		`{"email": "foo@bar.com", "password": "123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890"}`,

		// missing email
		`{password": "abracadabra"}`,
		// email too long
		`{"email": "123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890123457890@bar.com", "password": "abracadabra"}`,
	}

	for i, ex := range examples {
		t.Log("Example", i)

		rec := httptest.NewRecorder()
		req := &http.Request{Body: ioutil.NopCloser(bytes.NewBufferString(ex))}

		createUser(context.Background(), rec, req)

		if rec.Code != 400 {
			t.Errorf("status = %v want 400", rec.Code)
		}
	}
}
