package core

import (
	"expvar"
	"net/http"
	"sync"
	"time"

	"chain/metrics"
)

var (
	latencyMu sync.RWMutex
	latencies = map[string]*metrics.RotatingLatency{}

	latencyExpvar = expvar.NewMap("latency")
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
		l := metrics.NewRotatingLatency(5, 100*time.Millisecond)
		latencies[req.URL.Path] = l
		latencyExpvar.Set(req.URL.Path, l)
		return l
	}
	return nil
}
