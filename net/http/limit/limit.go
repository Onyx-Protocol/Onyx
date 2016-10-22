// Package limit provides a rate limiting HTTP handler.
package limit

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type handler struct {
	next    http.Handler
	limited http.Handler
	f       func(*http.Request) string
	freq    rate.Limit
	burst   int

	bucketMu sync.Mutex // protects the following
	buckets  map[string]*rate.Limiter
}

func Handler(next, limited http.Handler, freq, burst int, f func(*http.Request) string) http.Handler {
	return &handler{
		next:    next,
		limited: limited,
		f:       f,
		freq:    rate.Limit(freq),
		burst:   burst,
		buckets: make(map[string]*rate.Limiter),
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := h.f(r)
	if !h.bucket(id).Allow() {
		h.limited.ServeHTTP(w, r)
		return
	}
	h.next.ServeHTTP(w, r)
}

func (h *handler) bucket(id string) *rate.Limiter {
	h.bucketMu.Lock()
	bucket, ok := h.buckets[id]
	if !ok {
		bucket = rate.NewLimiter(h.freq, h.burst)
		h.buckets[id] = bucket
	}
	h.bucketMu.Unlock()
	return bucket
}

func RemoteAddrID(r *http.Request) string {
	return r.RemoteAddr
}

func AuthUserID(r *http.Request) string {
	user, _, _ := r.BasicAuth()
	return user
}
