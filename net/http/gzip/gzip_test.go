package gzip

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type noOpWriter struct{ header http.Header }

func (n noOpWriter) Header() http.Header {
	return n.header
}

func (n noOpWriter) Write(d []byte) (int, error) {
	return len(d), nil
}

func (n noOpWriter) WriteHeader(int) {}

func BenchmarkGzip(b *testing.B) {
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	w := noOpWriter{header: http.Header{}}

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(w, r)
	}
}

func TestGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTP(w, r)
	if s := w.HeaderMap.Get("content-encoding"); s != "gzip" {
		t.Errorf(`w.HeaderMap.Get("content-encoding") = %s want gzip`, s)
	}
}

func TestNoGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTP(w, r)
	if w.HeaderMap.Get("content-encoding") == "gzip" {
		t.Error("unexpected gzip")
	}
}
