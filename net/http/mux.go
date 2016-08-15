package http

import (
	"net/http"

	"golang.org/x/net/context"
)

// ServeMux is like http.ServeMux but it uses Handler instead of http.Handler.
// That is, it requires and includes context with each request.
type ServeMux struct {
	m *http.ServeMux
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{http.NewServeMux()} }

// Handle registers handler in m under the given pattern.
func (m *ServeMux) Handle(pattern string, handler Handler) {
	m.m.Handle(pattern, panicHandler{handler})
}

// ServeHTTPContext satisfies the Handler interface.
func (m *ServeMux) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	h, _ := m.m.Handler(req)
	if c, ok := h.(Handler); ok {
		c.ServeHTTPContext(ctx, w, req)
	} else {
		h.ServeHTTP(w, req)
	}
}

type panicHandler struct{ Handler }

func (panicHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	panic("chainhttp: bad handler") // can't happen; see ServeHTTPContext
}
