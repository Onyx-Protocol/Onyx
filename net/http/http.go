package http

import (
	"net/http"

	"github.com/tessr/pat"
	"golang.org/x/net/context"

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

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("HandlerFunc was called without context")
}

type ServeMux struct {
	*http.ServeMux
}

func (mux ServeMux) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h, _ := mux.Handler(r)
	if contextHandler, ok := h.(Handler); ok {
		contextHandler.ServeHTTPContext(ctx, w, r)
	} else {
		h.ServeHTTP(w, r)
	}
}

type PatServeMux struct {
	*pat.PatternServeMux
}

func (mux PatServeMux) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h := mux.Handler(r)
	if contextHandler, ok := h.(Handler); ok {
		contextHandler.ServeHTTPContext(ctx, w, r)
	} else {
		h.ServeHTTP(w, r)
	}
}

func (mux PatServeMux) AddFunc(method, pattern string, f HandlerFunc) {
	mux.Add(method, pattern, f)
}

// BackgroundHandler converts a Handler to an http.Handler
// by adding a new request ID to the background context.
type BackgroundHandler struct {
	Handler Handler
}

func (b BackgroundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(kr): take half of request ID from the client
	ctx := context.Background()
	ctx = reqid.NewContext(ctx, reqid.New())
	w.Header().Add("Chain-Request-Id", reqid.FromContext(ctx))
	b.Handler.ServeHTTPContext(ctx, w, r)
}
