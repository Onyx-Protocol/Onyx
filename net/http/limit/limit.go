package limit

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type BucketLimiter struct {
	freq  rate.Limit
	burst int

	bucketMu sync.Mutex // protects the following
	buckets  map[string]*rate.Limiter
}

func NewBucketLimiter(freq, burst int) *BucketLimiter {
	return &BucketLimiter{
		freq:    rate.Limit(freq),
		burst:   burst,
		buckets: make(map[string]*rate.Limiter),
	}
}

func (b *BucketLimiter) Allow(id string) bool {
	return b.bucket(id).Allow()
}

func (b *BucketLimiter) bucket(id string) *rate.Limiter {
	b.bucketMu.Lock()
	bucket, ok := b.buckets[id]
	if !ok {
		bucket = rate.NewLimiter(b.freq, b.burst)
		b.buckets[id] = bucket
	}
	b.bucketMu.Unlock()
	return bucket
}

type handler struct {
	next    http.Handler
	limited http.Handler
	f       func(*http.Request) string

	limiter *BucketLimiter
}

func Handler(next, limited http.Handler, freq, burst int, f func(*http.Request) string) http.Handler {
	return &handler{
		next:    next,
		limited: limited,
		f:       f,
		limiter: NewBucketLimiter(freq, burst),
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := h.f(r)
	if !h.limiter.Allow(id) {
		h.limited.ServeHTTP(w, r)
		return
	}
	h.next.ServeHTTP(w, r)
}

func RemoteAddrID(r *http.Request) string {
	return r.RemoteAddr
}

func AuthUserID(r *http.Request) string {
	user, _, _ := r.BasicAuth()
	return user
}
