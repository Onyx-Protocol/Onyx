// Package metrics provides convenient facilities to record
// on-line high-level performance metrics.
package metrics

import (
	"bytes"
	"encoding/json"
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/codahale/hdrhistogram"
)

// Period is the size of a RotatingLatency bucket.
// Each RotatingLatency will rotate once per Period.
const Period = time.Minute

var (
	rotatingLatenciesMu sync.Mutex
	rotatingLatencies   []*RotatingLatency
	latencyExpvar       = expvar.NewMap("latency")
)

// PublishLatency publishes rl as an expvar inside the
// global latency map (which is itself published under
// the key "latency").
func PublishLatency(key string, rl *RotatingLatency) {
	latencyExpvar.Set(key, rl)
}

// A Latency records information about the aggregate latency
// of an operation over time.
// Internally it holds an HDR histogram (to three significant figures)
// and a counter of attempts to record a value
// greater than the histogram's max.
type Latency struct {
	limit time.Duration // readonly

	time  time.Time
	hdr   hdrhistogram.Histogram
	nover int           // how many values were over limit
	max   time.Duration // max recorded value (can be over limit)
}

// NewLatency returns a new latency histogram with the given
// duration limit and with three significant figures of precision.
func NewLatency(limit time.Duration) *Latency {
	return &Latency{
		hdr:   *hdrhistogram.New(0, int64(limit), 2),
		limit: limit,
	}
}

// Record attempts to record a duration in the histogram.
// If d is greater than the max allowed duration,
// it increments a counter instead.
func (l *Latency) Record(d time.Duration) {
	if d > l.max {
		l.max = d
	}
	if d > l.limit {
		l.nover++
	} else {
		l.hdr.RecordValue(int64(d))
	}
}

// Reset resets l to is original empty state.
func (l *Latency) Reset() {
	l.hdr.Reset()
	l.nover = 0
}

// String returns l as a JSON string.
// This makes it suitable for use as an expvar.Val.
func (l *Latency) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, `{"Histogram":`)
	h, _ := json.Marshal((&l.hdr).Export()) // #nosec
	b.Write(h)
	fmt.Fprintf(&b, `,"Over":%d,"Timestamp":%d,"Max":%d}`, l.nover, l.time.Unix(), l.max)
	return b.String()
}

// A RotatingLatency holds a rotating circular buffer of Latency objects,
// that rotates once per Period time.
// It can be used as an expvar Val.
// Its exported methods are safe to call concurrently.
type RotatingLatency struct {
	mu  sync.Mutex
	l   []Latency
	n   int
	cur *Latency
}

// NewRotatingLatency returns a new rotating latency recorder
// with n buckets of history.
func NewRotatingLatency(n int, max time.Duration) *RotatingLatency {
	r := &RotatingLatency{
		l: make([]Latency, n),
	}
	for i := range r.l {
		r.l[i] = *NewLatency(max)
	}
	r.rotate()
	rotatingLatenciesMu.Lock()
	rotatingLatencies = append(rotatingLatencies, r)
	rotatingLatenciesMu.Unlock()
	return r
}

// Record attempts to record a duration in the current Latency in r.
// If d is greater than the max allowed duration,
// it increments a counter instead.
func (r *RotatingLatency) Record(d time.Duration) {
	r.mu.Lock()
	r.cur.Record(d)
	r.mu.Unlock()
}

func (r *RotatingLatency) RecordSince(t0 time.Time) {
	r.Record(time.Since(t0))
}

func (r *RotatingLatency) rotate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cur != nil {
		r.cur.time = time.Now()
	}
	r.n++
	r.cur = &r.l[r.n%len(r.l)]
	r.cur.Reset()
}

// String returns r as a JSON string.
// This makes it suitable for use as an expvar.Val.
//
// Example:
//
//  {
//      "NumRot": 204,
//      "Buckets": [
//          {
//              "Over": 4,
//              "Histogram": {
//                  "LowestTrackableValue": 0,
//                  "HighestTrackableValue": 1000000000,
//                  "SignificantFigures": 2,
//                  "Counts": [2,0,15,...]
//              }
//          },
//          ...
//      ]
//  }
//
// Note that the last bucket is actively recording values.
// To collect complete and accurate data over a long time,
// store the next-to-last bucket after each rotation.
// The last bucket is only useful for a "live" view
// with finer granularity than the rotation period (which is one minute).
func (r *RotatingLatency) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b bytes.Buffer
	fmt.Fprintf(&b, `{"Buckets":[`)
	for i := range r.l {
		if i > 0 {
			b.WriteByte(',')
		}
		j := (r.n + i + 1) % len(r.l)
		fmt.Fprintf(&b, "%s", &r.l[j])
	}
	fmt.Fprintf(&b, `],"NumRot":%d}`, r.n)
	return b.String()
}

func init() {
	go func() {
		for range time.Tick(Period) {
			rotatingLatenciesMu.Lock()
			a := rotatingLatencies
			rotatingLatenciesMu.Unlock()
			for _, rot := range a {
				rot.rotate()
			}
		}
	}()
}
