package bc

// TimeRange denotes a time range. It satisfies the Entry interface.
type TimeRange struct {
	body struct {
		MinTimeMS, MaxTimeMS uint64
		ExtHash              Hash
	}
}

func (TimeRange) Type() string          { return "timerange1" }
func (tr *TimeRange) Body() interface{} { return tr.body }

func (TimeRange) Ordinal() int { return -1 }

// NewTimeRange creates a new TimeRange.
func NewTimeRange(minTimeMS, maxTimeMS uint64) *TimeRange {
	tr := new(TimeRange)
	tr.body.MinTimeMS = minTimeMS
	tr.body.MaxTimeMS = maxTimeMS
	return tr
}
