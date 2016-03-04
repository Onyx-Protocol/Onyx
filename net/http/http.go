package http

import (
	"net/http"
	"runtime"

	"golang.org/x/net/context"

	"chain/log"
	"chain/net/http/reqid"
)

type Handler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request)
}

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTPContext calls f(ctx, w, r).
func (f HandlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

type noContextHandler struct {
	h http.Handler
}

// DropContext returns a Handler that ignores its context
// and simply calls h.
func DropContext(h http.Handler) Handler {
	return noContextHandler{h}
}

func (h noContextHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) { h.h.ServeHTTP(w, req) }
func (h noContextHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	h.h.ServeHTTP(w, req)
}

// ContextHandler converts a Handler to an http.Handler
// by adding a new request ID to the given context.
type ContextHandler struct {
	Context context.Context
	Handler Handler
}

func (b ContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(kr): take half of request ID from the client
	ctx := b.Context
	ctx = reqid.NewContext(ctx, reqid.New())
	defer func() {
		if err := recover(); err != nil {
			// See also $GOROOT/src/net/http/server.go.
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Write(ctx,
				log.KeyMessage, "panic",
				"remote-addr", r.RemoteAddr,
				log.KeyError, err,
				log.KeyStack, buf,
			)
		}
	}()
	w.Header().Add("Chain-Request-Id", reqid.FromContext(ctx))
	b.Handler.ServeHTTPContext(ctx, w, r)
}
