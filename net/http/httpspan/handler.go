package httpspan

import (
	"net/http"
	"strings"

	"github.com/resonancelabs/go-pub/instrument"
	"golang.org/x/net/context"

	"chain/log"
	chainhttp "chain/net/http"
	"chain/net/http/reqid"
	"chain/net/trace/span"
)

const (
	fieldJoinID   = "Trace-Join-Id"   // canonical mime header key form
	fieldParentID = "Trace-Parent-Id" // canonical mime header key form
)

// Handler starts a span for the current HTTP request
// and adds it to the context
// before calling the underlying handler.
// It uses HTTP header fields to link the span
// into a trace identified by the client.
type Handler struct {
	Handler chainhttp.Handler
}

// ServeHTTPContext satisfies the http.Handler interface.
// It expects to find a request ID in the context.
func (h Handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	sp := instrument.StartSpan()
	sp.SetOperation("http.request")
	if s := req.Header.Get(fieldParentID); s != "" {
		sp.AddAttribute("parent_span_guid", s)
	}
	for _, kv := range req.Header[fieldJoinID] {
		if i := strings.Index(kv, "="); i >= 0 {
			sp.AddTraceJoinId(kv[:i], kv[i+1:])
		}
	}
	sp.AddTraceJoinId("reqid", reqid.FromContext(ctx))
	ctx = span.NewContextWithSpan(ctx, sp)
	log.Write(
		ctx,
		"user-agent", req.Header.Get("User-Agent"),
		"path", req.URL.Path,
	)
	h.Handler.ServeHTTPContext(ctx, w, req)
	sp.Finish()
}
