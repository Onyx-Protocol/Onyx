package tx

type timeRange struct {
	body struct {
		MinTimeMS, MaxTimeMS uint64
		ExtHash              extHash
	}
}

func (timeRange) Type() string          { return "timerange" } // xxx "timerange1"?
func (tr *timeRange) Body() interface{} { return tr.body }

func (timeRange) Ordinal() int { return -1 }

func newTimeRange(minTimeMS, maxTimeMS uint64) *timeRange {
	tr := new(timeRange)
	tr.body.MinTimeMS = minTimeMS
	tr.body.MaxTimeMS = maxTimeMS
	return tr
}
