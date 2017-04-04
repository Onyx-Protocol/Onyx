package bc

// TimeRange denotes a time range. It satisfies the Entry interface.

func (TimeRange) Type() string          { return "timerange1" }
func (tr *TimeRange) body() interface{} { return tr.Body }

// NewTimeRange creates a new TimeRange.
func NewTimeRange(minTimeMS, maxTimeMS uint64) *TimeRange {
	tr := new(TimeRange)
	tr.Body.MinTimeMs = minTimeMS
	tr.Body.MaxTimeMs = maxTimeMS
	return tr
}
