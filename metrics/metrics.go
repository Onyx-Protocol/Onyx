// Package metrics provides metrics-related utilities.
// Defined metrics:
//   requests (counter)
//   respcode.200 (counter)
//   respcode.404 (counter)
//   respcode.NNN (etc)
package metrics

import (
	chainhttp "chain/net/http"
	"net/http"
	"strconv"

	"github.com/codahale/metrics"
	"golang.org/x/net/context"
)

// Handler counts requests and response codes in metrics.
// See the package doc for metric names.
type Handler struct {
	Handler chainhttp.Handler
}

func (h Handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// TODO(kr): generate counters automatically based on path
	metrics.Counter("requests").Add()
	h.Handler.ServeHTTPContext(ctx, &codeCountResponse{w, false}, req)
}

type codeCountResponse struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *codeCountResponse) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	metrics.Counter("respcode." + strconv.Itoa(code)).Add()
	w.ResponseWriter.WriteHeader(code)
}

func (w *codeCountResponse) Write(p []byte) (int, error) {
	w.WriteHeader(200)
	return w.ResponseWriter.Write(p)
}
