package core

import (
	"expvar"
	"net/http"
	"sync"
	"time"

	"chain/metrics"
)

var (
	latencyMu sync.Mutex
	latencies = map[string]*metrics.RotatingLatency{}

	latencyRange = map[string]time.Duration{
		crosscoreRPCPrefix + "get-block":         20 * time.Second,
		crosscoreRPCPrefix + "signer/sign-block": 5 * time.Second,
		crosscoreRPCPrefix + "get-snapshot":      30 * time.Second,
		// the rest have a default range
	}
)

// latency returns a rotating latency histogram for the given request.
func latency(tab *http.ServeMux, req *http.Request) *metrics.RotatingLatency {
	latencyMu.Lock()
	defer latencyMu.Unlock()
	if l := latencies[req.URL.Path]; l != nil {
		return l
	}
	// Create a histogram only if the path is legit.
	if _, pat := tab.Handler(req); pat == req.URL.Path {
		d, ok := latencyRange[req.URL.Path]
		if !ok {
			d = 100 * time.Millisecond
		}
		l := metrics.NewRotatingLatency(5, d)
		latencies[req.URL.Path] = l
		metrics.PublishLatency(req.URL.Path, l)
		return l
	}
	return nil
}

var (
	ncoreMu   sync.Mutex
	ncore     = expvar.NewInt("ncore")
	ncoreTime time.Time
	coresSeen map[string]bool
)

func coreCounter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if coreID := req.Header.Get("Chain-Core-ID"); coreID != "" {
			countCore(coreID)
		}
		h.ServeHTTP(w, req)
	})
}

func countCore(id string) {
	t := time.Now()
	ncoreMu.Lock()
	defer ncoreMu.Unlock()
	if t.Sub(ncoreTime) > time.Minute {
		ncore.Set(int64(len(coresSeen)))
		ncoreTime = t
		coresSeen = make(map[string]bool)
	}
	coresSeen[id] = true
}
