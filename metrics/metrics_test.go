package metrics

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/codahale/hdrhistogram"
)

func TestRotString(t *testing.T) {
	rot := NewRotatingLatency(1, time.Second)
	rot.l[0].hdr = *hdrhistogram.New(0, int64(time.Second), 1)

	want := `{
		"NumRot": 1,
		"Buckets": [{
			"Over": 0,
			"Histogram": {
				"LowestTrackableValue": 0,
				"HighestTrackableValue": 1000000000,
				"SignificantFigures": 1,
				"Counts": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
			}
		}]
	}`

	got := rot.String()
	if !jsonIsEqual(t, got, want) {
		t.Errorf("%#v.String() = %#q want %#q", rot, got, want)
	}
}

func jsonIsEqual(t *testing.T, a, b string) bool {
	var av, bv interface{}

	err := json.Unmarshal([]byte(a), &av)
	if err != nil {
		t.Fatal(err, a)
	}
	err = json.Unmarshal([]byte(b), &bv)
	if err != nil {
		t.Fatal(err, b)
	}

	return reflect.DeepEqual(av, bv)
}
