// Package reqid creates request IDs and stores them in Contexts.
package reqid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"chain/log"
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
)

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
		log.Printf(context.Background(), "error making reqID")
	}
	return hex.EncodeToString(b)
}

// NewContext returns a new Context that carries reqid.
// It also adds a log prefix to print the request ID using
// package chain/log.
func NewContext(ctx context.Context, reqid string) context.Context {
	ctx = context.WithValue(ctx, reqIDKey, reqid)
	ctx = log.AddPrefixkv(ctx, "reqid", reqid)
	return ctx
}

// FromContext returns the request ID stored in ctx,
// if any.
func FromContext(ctx context.Context) string {
	reqID, _ := ctx.Value(reqIDKey).(string)
	return reqID
}

// NewSubContext returns a new Context that carries subreqid
// It also adds a log prefix to print the sub-request ID using
// package chain/log.
func NewSubContext(ctx context.Context, reqid string) context.Context {
	ctx = context.WithValue(ctx, subReqIDKey, reqid)
	ctx = log.AddPrefixkv(ctx, "subreqid", reqid)
	return ctx
}

// FromSubContext returns the sub-request ID stored in ctx,
// if any.
func FromSubContext(ctx context.Context) string {
	subReqID, _ := ctx.Value(subReqIDKey).(string)
	return subReqID
}

func Handler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		// TODO(kr): take half of request ID from the client
		id := New()
		ctx = NewContext(ctx, id)

		defer func() {
			if err := recover(); err != nil {
				log.Printkv(ctx,
					"message", "panic",
					"remote-addr", req.RemoteAddr,
					"error", err,
				)
			}
		}()
		w.Header().Add("Chain-Request-Id", id)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}
