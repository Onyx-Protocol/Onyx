package metrics

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func BenchmarkRecord(b *testing.B) {
	rot := NewRotatingLatency(5, time.Second)
	for i := 0; i < b.N; i++ {
		rot.Record(0)
	}
}

func BenchmarkStringFormat(b *testing.B) {
	rot := NewRotatingLatency(5, time.Second)
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(ioutil.Discard, "%s", rot)
	}
}
