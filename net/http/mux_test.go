package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

func TestHandler(t *testing.T) {
	m := NewServeMux()
	m.Handle("/foo", testHandler{})

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	m.ServeHTTPContext(context.Background(), resp, req)
	if resp.Code != 200 {
		t.Errorf("response = %d want 200", resp.Code)
	}
}

func TestNotFound(t *testing.T) {
	m := NewServeMux()
	m.Handle("/foo", testHandler{})

	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://example.com/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	m.ServeHTTPContext(context.Background(), resp, req)
	if resp.Code != 404 {
		t.Errorf("response = %d want 404", resp.Code)
	}
}

type testHandler struct{}

func (testHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, `"ok"`+"\n")
}
