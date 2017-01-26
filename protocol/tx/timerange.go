package tx

type timeRange struct {
	body struct {
		minTime int // TODO: determine kind of int
		maxTime int
		extHash extHash
	}
}

func (timeRange) Type() string          { return "timerange" }
func (tr *timeRange) Body() interface{} { return tr.body }

func newTimeRange(mintime, maxtime int) *timeRange {
	tr := new(timeRange)
	tr.body.minTime = mintime
	tr.body.maxTime = maxtime
	return tr
}
