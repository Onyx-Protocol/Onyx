package httpjson

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

func TestMuxOk(t *testing.T) {
	errFunc := func(ctx context.Context, w http.ResponseWriter, err error) {}
	m := NewServeMux(errFunc)
	m.HandleFunc("POST", "/", func(a string) string { return a })

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`"2"`))
	m.ServeHTTPContext(nil, resp, req)
	if resp.Code != 200 {
		t.Errorf("response code = %d want 200", resp.Code)
	}
	got := strings.TrimSpace(resp.Body.String())
	want := `"2"`
	if got != want {
		t.Errorf("response body = %#q want %#q", got, want)
	}
}

func TestMuxErr(t *testing.T) {
	defer func() {
		got := recover()
		if got == nil {
			t.Error("err = nil want non-nil error")
		}
	}()

	m := NewServeMux(nil)
	m.HandleFunc("GET", "/", "not-a-func")
}
