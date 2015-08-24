package http

import (
	"net/http"

	"github.com/tessr/pat"
	"golang.org/x/net/context"
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
