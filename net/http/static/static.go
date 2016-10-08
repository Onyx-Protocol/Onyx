// Package static provides a handler for serving static assets from an in-memory
// map.
package static

import (
	"net/http"
	"strings"
	"time"
)

// use start time as a conservative bound for last-modified
var lastMod = time.Now()

type Handler struct {
	Assets map[string]string
	Index  string
}

func (h Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	output, ok := h.Assets[r.URL.Path]
	if !ok && h.Index == "" {
		http.NotFound(rw, r)
		return
	}
	if !ok {
		output = h.Assets[h.Index]
	}

	http.ServeContent(rw, r, r.URL.Path, lastMod, strings.NewReader(output))
}
