package httpjson

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContext(t *testing.T) {
	wantHead := "bar"
	wantRespHead := "baz"
	f := func(ctx context.Context) {
		if g := Request(ctx).Header.Get("Test-Key"); g != wantHead {
			t.Errorf("header = %q want %q", g, wantHead)
		}
		ResponseWriter(ctx).Header().Set("Test-Resp-Key", wantRespHead)
	}

	h, err := Handler(f, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Test-Key", wantHead)
	h.ServeHTTP(resp, req)
	if g := resp.Header().Get("Test-Resp-Key"); g != wantRespHead {
		t.Errorf("header = %q want %q", g, wantRespHead)
	}
}
