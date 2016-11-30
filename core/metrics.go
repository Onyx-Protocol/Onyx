package core

import (
	"expvar"
	"sync"
	"time"

	"chain/metrics"
)

var (
	latencyMu sync.Mutex
	latencies = map[string]*metrics.RotatingLatency{}

	latencyRange = map[string]time.Duration{
		"pb.Network.GetBlock":    20 * time.Second,
		"pb.Network.GetSnapshot": 30 * time.Second,
		"pb.Signer.SignBlock":    5 * time.Second,
		// the rest have a default range
	}
)

// latency returns a rotating latency histogram for the given request.
func latency(m string) *metrics.RotatingLatency {
	latencyMu.Lock()
	defer latencyMu.Unlock()
	if l := latencies[m]; l != nil {
		return l
	}

	d, ok := latencyRange[m]
	if !ok {
		d = 100 * time.Millisecond
	}
	l := metrics.NewRotatingLatency(5, d)
	latencies[m] = l
	metrics.PublishLatency(m, l)
	return l
}

var (
	ncoreMu   sync.Mutex
	ncore     = expvar.NewInt("ncore")
	ncoreTime time.Time
	coresSeen map[string]bool
)

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
