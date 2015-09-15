package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"chain/net/http/pat"
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
	w.Header().Add("Chain-Request-Id", reqid.FromContext(ctx))
	b.Handler.ServeHTTPContext(ctx, w, r)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ListenAndServeTLS is like the same function in net/http except
// cert and key are the actual cert and key instead of files containing
// the cert and key.
func ListenAndServeTLS(addr, cert, key string, handler http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: handler}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return srv.Serve(tlsListener)
}
