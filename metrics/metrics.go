// Package metrics provides metrics-related utilities.
// Defined metrics:
//   requests (counter)
//   respcode.200 (counter)
//   respcode.404 (counter)
//   respcode.NNN (etc)
package metrics

import (
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codahale/metrics"
	_ "github.com/codahale/metrics/runtime"
	"golang.org/x/net/context"

	"chain/log"
	chainhttp "chain/net/http"
)

var (
	// MaxDuration is the maximum value of histograms
	// that are created through RecordElapsed
	MaxDuration = 5 * time.Second

	// SigFigs is the number of significant figures for histograms
	// that are created through RecordElapsed
	SigFigs = 3

	hm         sync.Mutex // protects the following
	histograms = make(map[*runtime.Func]*metrics.Histogram)

	gcpause = metrics.NewHistogram("Mem.GCPauseTime.duration", 0, (10 * time.Second).Nanoseconds(), 3)
	nrange  = metrics.Counter("Histogram.RangeErr")
)

// Handler counts requests and response codes in metrics.
// See the package doc for metric names.
type Handler struct {
	Handler chainhttp.Handler
}

// ServeHTTPContext satisfies chainhttp.Handler interface
func (h Handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// TODO(kr): generate counters automatically based on path
	metrics.Counter("requests").Add()
	h.Handler.ServeHTTPContext(ctx, &codeCountResponse{w, false}, req)
}

type codeCountResponse struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *codeCountResponse) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	metrics.Counter("respcode." + strconv.Itoa(code)).Add()
	w.ResponseWriter.WriteHeader(code)
}

func (w *codeCountResponse) Write(p []byte) (int, error) {
	w.WriteHeader(200)
	return w.ResponseWriter.Write(p)
}

func normalizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '(' || r == ')' || r == '*' {
			return -1
		}
		return r
	}, name[strings.LastIndex(name, "/")+1:])
}

func histogram(pc uintptr) *metrics.Histogram {
	hm.Lock()

	f := runtime.FuncForPC(pc)
	hist, ok := histograms[f]
	if !ok {
		name := normalizeName(f.Name() + ".duration")
		hist = metrics.NewHistogram(name, 0, MaxDuration.Nanoseconds(), SigFigs)
		histograms[f] = hist
	}

	hm.Unlock()

	return hist
}

// RecordElapsed records the time elapsed since t0
// on a histogram named after the caller.
// It should be called at most one time per function,
// since otherwise the results from multiple sources
// will be combined into one histogram.
//
// Example:
// 		defer metrics.RecordElapsed(time.Now())
func RecordElapsed(t0 time.Time) {
	elapsed := time.Since(t0)
	var pc [1]uintptr
	runtime.Callers(2, pc[:])

	err := histogram(pc[0]).RecordValue(elapsed.Nanoseconds())
	if err != nil {
		nrange.Add()
		log.Error(context.Background(), err)
	}
}

// Function recordGC polls the mem stats
// and records any unrecorded GC pause times
// in the HDR histogram gcpause.
// See also runtime.MemStats.
func recordGC(period time.Duration) {
	var (
		igc uint32
		m   runtime.MemStats
	)
	for range time.Tick(period) {
		runtime.ReadMemStats(&m)
		for ; igc < m.NumGC; igc++ {
			err := gcpause.RecordValue(int64(m.PauseNs[igc%256]))
			if err != nil {
				log.Error(context.Background(), err)
			}
		}
	}
}

func init() {
	go recordGC(time.Second)
}
