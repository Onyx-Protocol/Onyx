package gzip

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

type handlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f handlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	f(ctx, w, req)
}

func TestGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	ctx := context.Background()
	h := Handler{handlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTPContext(ctx, w, r)
	if s := w.HeaderMap.Get("content-encoding"); s != "gzip" {
		t.Errorf(`w.HeaderMap.Get("content-encoding") = %s want gzip`, s)
	}
}

func TestNoGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	ctx := context.Background()
	h := Handler{handlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTPContext(ctx, w, r)
	if w.HeaderMap.Get("content-encoding") == "gzip" {
		t.Error("unexpected gzip")
	}
}
