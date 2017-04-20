package bc

import "io"

// TimeRange denotes a time range. It satisfies the Entry interface.

func (TimeRange) typ() string { return "timerange1" }
func (tr *TimeRange) writeForHash(w io.Writer) {
	mustWriteForHash(w, tr.MinTimeMs)
	mustWriteForHash(w, tr.MaxTimeMs)
	mustWriteForHash(w, tr.ExtHash)
}

// NewTimeRange creates a new TimeRange.
func NewTimeRange(minTimeMS, maxTimeMS uint64) *TimeRange {
	return &TimeRange{
		MinTimeMs: minTimeMS,
		MaxTimeMs: maxTimeMS,
	}
}
