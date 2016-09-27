package httpjson

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestUnmarshalDuration(t *testing.T) {
	successCases := []string{
		`1000`, // this is an "integer"
		`"1000ms"`,
		`"1000000000ns"`,
		`"1s"`,
	}

	for _, c := range successCases {
		var dur Duration
		err := json.Unmarshal([]byte(c), &dur)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		var want float64 = 1 // all of our inputs equal 1 second
		if got := dur.Seconds(); got != want {
			t.Errorf("Duration.UnmarshalJSON(%q) = %f want %f", c, got, want)
		}
	}

	negativeCases := []string{
		`-1000`,
		`"-1000ms"`,
	}

	for _, c := range negativeCases {
		var dur Duration
		wantErr := "invalid httpjson.Duration: Duration cannot be less than 0"
		err := json.Unmarshal([]byte(c), &dur)
		if err.Error() != wantErr {
			t.Errorf("wanted error %s, got %s", wantErr, err)
		}
	}
}

func TestMarshalDuration(t *testing.T) {
	dur := Duration{
		Duration: time.Second,
	}
	b, err := json.Marshal(dur)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	got, err := strconv.Atoi(string(b))
	if err != nil {
		t.Fatal(err)
	}
	want := 1000
	if got != want {
		t.Errorf("wanted %d, got %d", want, got)
	}
}
