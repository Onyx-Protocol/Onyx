package httpjson

import (
	"net/http"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
	"chain/net/http/pat"
)

// ErrorWriter is responsible for writing the provided error value
// to the response.
type ErrorWriter func(context.Context, http.ResponseWriter, error)

// ServeMux is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered patterns and calls the
// function for the pattern that most closely matches the URL.
// See package chain/net/http/pat for details
// of the pattern matching algorithm.
//
// Each function must have an appropriate signature.
// See the package doc for details.
type ServeMux struct {
	pat     chainhttp.PatServeMux
	errFunc ErrorWriter
}

// NewServeMux allocates and returns a new ServeMux.
// Handlers in the returned ServeMux will call f
// when the handler function returns an error.
func NewServeMux(f ErrorWriter) *ServeMux {
	m := new(ServeMux)
	m.pat.PatternServeMux = pat.New()
	m.errFunc = f
	return m
}

// ServeHTTPContext dispatches the request to the handler
// whose pattern most closely matches the request URL.
func (m *ServeMux) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	m.pat.ServeHTTPContext(ctx, w, req)
}

// HandleFunc adds f to the handler table in m.
// If f is not a function with a compatible signature,
// HandleFunc panics.
func (m *ServeMux) HandleFunc(method, pattern string, f interface{}) {
	h, err := newHandler(pattern, f, m.errFunc)
	if err != nil {
		panic(err)
	}
	m.pat.AddFunc(method, pattern, h.ServeHTTPContext)
}
