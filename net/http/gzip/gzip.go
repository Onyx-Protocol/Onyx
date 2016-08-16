package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

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
	gz := gzip.NewWriter(w)
	type (
		response     struct{ http.ResponseWriter }
		gzipResponse struct {
			io.Writer
			response
		}
	)
	h.Handler.ServeHTTP(gzipResponse{gz, response{w}}, r)
	gz.Close()
}
