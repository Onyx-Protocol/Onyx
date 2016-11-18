package gzip

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed) // #nosec
		return w
	},
}

func getWriter(w io.Writer) *gzip.Writer {
	gz := pool.Get().(*gzip.Writer)
	gz.Reset(w)
	return gz
}

type Handler struct {
	Handler http.Handler
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "Accept-Encoding")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		h.Handler.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := getWriter(w)
	w = &responseWriter{gz, w}
	h.Handler.ServeHTTP(w, r)
	gz.Close()
	pool.Put(gz)
}

type responseWriter struct {
	w                   io.Writer // w wraps only method Write
	http.ResponseWriter           // embedded for the other methods
}

var _ http.ResponseWriter = (*responseWriter)(nil)
var _ http.Hijacker = (*responseWriter)(nil)

func (w *responseWriter) Write(p []byte) (int, error) { return w.w.Write(p) }

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("not a hijacker")
	}
	return h.Hijack()
}
