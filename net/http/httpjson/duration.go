package httpjson

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

type Duration struct {
	time.Duration
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
// It attempts to parse text as a time.Duration string.
// The Go documentation defines this as a possibly signed sequence of decimal
// numbers, each with optional fraction and a unit suffix, such as
// "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h").
// If there is no time unit, UnmarshalText defaults to ms.
func (d *Duration) UnmarshalJSON(b []byte) error {
	dMS, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		// Assume this is a string instead, in which case we need to unmarshal it as a string
		// before we try to parse it as a time.Duration.
		var str string
		err = json.Unmarshal(b, &str)
		if err != nil {
			return errors.New("invalid httpjson.Duration")
		}

		d0, err := time.ParseDuration(str)
		if err != nil {
			return errors.New("invalid httpjson.Duration")
		}
		if d0 < 0 {
			return errors.New("invalid httpjson.Duration: Duration cannot be less than 0")
		}
		d.Duration = d0
	} else {
		if dMS < 0 {
			return errors.New("invalid httpjson.Duration: Duration cannot be less than 0")
		}
		d.Duration = time.Duration(dMS) * time.Millisecond
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.Nanoseconds() / int64(time.Millisecond))
}
