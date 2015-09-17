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

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		start, want string
	}{
		{"chain/api/asset.transfer.duration", "asset.transfer.duration"},
		{"chain/api/appdb.(*Address).Insert.duration", "appdb.Address.Insert.duration"},
	}

	for _, c := range cases {
		got := normalizeName(c.start)

		if got != c.want {
			t.Errorf("got normalizeName(%q) = %q want %q", c.start, got, c.want)
		}
	}
}
