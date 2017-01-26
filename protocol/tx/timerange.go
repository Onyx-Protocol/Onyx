package tx

type timeRange struct {
	body struct {
		minTimeMS, maxTimeMS uint64
		extHash              extHash
	}
}

func (timeRange) Type() string          { return "timerange" } // xxx "timerange1"?
func (tr *timeRange) Body() interface{} { return tr.body }

func newTimeRange(minTimeMS, maxTimeMS uint64) *timeRange {
	tr := new(timeRange)
	tr.body.minTimeMS = minTimeMS
	tr.body.maxTimeMS = maxTimeMS
	return tr
}
