package core

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"chain/database/pg"
	"chain/errors"
)

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		json string
		code int
	}{
		{nil, `{"code":"CH000","message":"Chain API Error","temporary":true}`, 500},
		{pg.ErrUserInputNotFound, `{"code":"CH002","message":"Not found","temporary":false}`, 400},
		{errors.Wrap(pg.ErrUserInputNotFound, "foo"), `{"code":"CH002","message":"Not found","temporary":false}`, 400},
		{errors.WithDetail(pg.ErrUserInputNotFound, "foo"), `{"code":"CH002","message":"Not found","detail":"foo","temporary":false}`, 400},
		{context.DeadlineExceeded, `{"code":"CH001","message":"Request timed out","temporary":true}`, 408},
	}

	for _, test := range cases {
		resp := httptest.NewRecorder()
		errorFormatter.Write(context.Background(), resp, test.err)
		got := strings.TrimSpace(resp.Body.String())
		if got != test.json {
			t.Errorf("writeHTTPError(%#v) wrote %s want %s", test.err, got, test.json)
		}
		if resp.Code != test.code {
			t.Errorf("writeHTTPError(%#v) wrote status %d want %d", test.err, resp.Code, test.code)
		}
	}
}
