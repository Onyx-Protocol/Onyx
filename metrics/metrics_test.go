package metrics

import (
	"testing"
	"time"
)

func BenchmarkRecordElapsed(b *testing.B) {
	t := time.Now()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		RecordElapsed(t)
	}
}
