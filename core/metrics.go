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

	latencyExpvar = expvar.NewMap("latency")
	latencyRange  = map[string]time.Duration{
		networkRPCPrefix + "get-block":         5 * time.Second,
		networkRPCPrefix + "get-blocks":        5 * time.Second,
		networkRPCPrefix + "signer/sign-block": 5 * time.Second,
		networkRPCPrefix + "get-snapshot":      30 * time.Second,
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
		latencyExpvar.Set(req.URL.Path, l)
		return l
	}
	return nil
}
