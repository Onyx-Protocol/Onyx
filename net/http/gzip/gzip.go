package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
)

type Handler struct {
	h chainhttp.Handler
}

func (h Handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "Accept-Encoding")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		h.h.ServeHTTPContext(ctx, w, r)
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
	h.h.ServeHTTPContext(ctx, gzipResponse{gz, response{w}}, r)
	gz.Close()
}
