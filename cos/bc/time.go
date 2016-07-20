package bc

import "time"

// NowMillis returns the current time in milliseconds,
// as defined by the protocol.
func NowMillis() uint64 {
	ms := time.Now().UnixNano() / int64(time.Millisecond)
	return uint64(ms)
}
