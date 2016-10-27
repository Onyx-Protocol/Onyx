// Package reqid creates request IDs and stores them in Contexts.
package reqid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"runtime"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

const (
	// reqIDKey is the key for request IDs in Contexts.  It is
	// unexported; clients use NewContext and FromContext
	// instead of using this key directly.
	reqIDKey key = iota
	// subReqIDKey is the key for sub-request IDs in Contexts.  It is
	// unexported; clients use NewSubContext and FromSubContext
	// instead of using this key directly.
	subReqIDKey
	// coreIDKey is the key for Chain-Core-ID request header field values.
	// It is only for statistics; don't use it for authorization.
	coreIDKey
	// pathKey is the key for the request path being handled.
	pathKey
)

const Unknown = "unknown_req_id"

// New generates a random request ID.
func New() string {
	// Given n IDs of length b bits, the probability that there will be a collision is bounded by
	// the number of pairs of IDs multiplied by the probability that any pair might collide:
	// p ≤ n(n - 1)/2 * 1/(2^b)
	//
	// We assume an upper bound of 1000 req/sec, which means that in a week there will be
	// n = 1000 * 604800 requests. If l = 10, b = 8*10, then p ≤ 1.512e-7, which is a suitably
	// low probability.
	l := 10
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("error making reqID")
	}
	return hex.EncodeToString(b)
}

// NewContext returns a new Context that carries reqid.
func NewContext(ctx context.Context, reqid string) context.Context {
	return context.WithValue(ctx, reqIDKey, reqid)
}

// FromContext returns the request ID stored in ctx,
// or Unknown, if there is none.
func FromContext(ctx context.Context) string {
	reqID, ok := ctx.Value(reqIDKey).(string)
	if !ok {
		return Unknown
	}
	return reqID
}

// CoreIDFromContext returns the Chain-Core-ID stored in ctx,
// or the empty string.
func CoreIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(coreIDKey).(string)
	return id
}

// PathFromContext returns the HTTP path stored in ctx,
// or the empty string.
func PathFromContext(ctx context.Context) string {
	path, _ := ctx.Value(pathKey).(string)
	return path
}

// NewSubContext returns a new Context that carries subreqid
func NewSubContext(ctx context.Context, reqid string) context.Context {
	return context.WithValue(ctx, subReqIDKey, reqid)
}

// FromSubContext returns the sub-request ID stored in ctx,
// or Unknown if there is none
func FromSubContext(ctx context.Context) string {
	subReqID, ok := ctx.Value(subReqIDKey).(string)
	if !ok {
		return Unknown
	}
	return subReqID
}

func Handler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		// TODO(kr): take half of request ID from the client
		id := New()
		ctx = NewContext(ctx, id)
		ctx = context.WithValue(ctx, coreIDKey, req.Header.Get("Chain-Core-ID"))
		ctx = context.WithValue(ctx, pathKey, req.URL.Path)
		defer func() {
			if err := recover(); err != nil {
				// See also $GOROOT/src/net/http/server.go.
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				// TODO(kr): use chain/log here
				// log.Write(ctx, log.KeyMessage, "panic", "remote-addr", req.RemoteAddr, log.KeyError, err, log.KeyStack, buf)
				log.Printf(`message=panic remote-addr=%q error=%q\n%s\n`, req.RemoteAddr, err, buf)
			}
		}()
		w.Header().Add("Chain-Request-Id", id)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}
