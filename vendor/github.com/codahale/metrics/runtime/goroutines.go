package runtime

import (
	"runtime"

	"github.com/codahale/metrics"
)

func init() {
	metrics.Gauge("Goroutines.Num").SetFunc(func() int64 {
		return int64(runtime.NumGoroutine())
	})
}
