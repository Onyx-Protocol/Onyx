package api

import (
	"chain/database/pg"
	"chain/errors"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

func TestWriteJSONArray(t *testing.T) {
	examples := []struct {
		in   []int
		want string
	}{
		{nil, "[]\n"},
		{[]int{}, "[]\n"},
		{make([]int, 0), "[]\n"},
	}

	for i, ex := range examples {
		t.Log("Example", i)
		rec := httptest.NewRecorder()
		writeJSON(context.Background(), rec, 200, ex.in)
		got, _ := ioutil.ReadAll(rec.Body)
		if string(got) != ex.want {
			t.Errorf("got=%v want=%v", string(got), ex.want)
		}
	}
}

func TestWriteHTTPError(t *testing.T) {
	cases := []struct {
		err  error
		json string
		code int
	}{
		{nil, `{"code":"CH000","message":"Chain API Error"}`, 500},
		{pg.ErrUserInputNotFound, `{"code":"CH005","message":"Not found."}`, 404},
		{errors.Wrap(pg.ErrUserInputNotFound, "foo"), `{"code":"CH005","message":"Not found."}`, 404},
		{errors.WithDetail(pg.ErrUserInputNotFound, "foo"), `{"code":"CH005","message":"Not found.","detail":"foo"}`, 404},
	}

	for _, test := range cases {
		resp := httptest.NewRecorder()
		writeHTTPError(context.Background(), resp, test.err)
		got := strings.TrimSpace(resp.Body.String())
		if got != test.json {
			t.Errorf("writeHTTPError(%#v) wrote %s want %s", test.err, got, test.json)
		}
		if resp.Code != test.code {
			t.Errorf("writeHTTPError(%#v) wrote status %d want %d", test.err, resp.Code, test.code)
		}
	}
}
