package bc

// TimeRange denotes a time range. It satisfies the Entry interface.
type TimeRange struct {
	Body struct {
		MinTimeMS, MaxTimeMS uint64
		ExtHash              Hash
	}
}

func (TimeRange) Type() string          { return "timerange1" }
func (tr *TimeRange) body() interface{} { return tr.Body }

func (TimeRange) Ordinal() int { return -1 }

// NewTimeRange creates a new TimeRange.
func NewTimeRange(minTimeMS, maxTimeMS uint64) *TimeRange {
	tr := new(TimeRange)
	tr.Body.MinTimeMS = minTimeMS
	tr.Body.MaxTimeMS = maxTimeMS
	return tr
}

func (tr *TimeRange) CheckValid(vs *validationState) error {
	if tr.Body.MinTimeMS > vs.tx.Body.MinTimeMS {
		return errBadTimeRange
	}
	if tr.Body.MaxTimeMS > 0 && tr.Body.MaxTimeMS < vs.tx.Body.MaxTimeMS {
		return errBadTimeRange
	}
	if vs.tx.Body.Version == 1 && (tr.Body.ExtHash != Hash{}) {
		return errNonemptyExtHash
	}
	return nil
}
