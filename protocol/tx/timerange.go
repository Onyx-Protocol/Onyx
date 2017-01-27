package tx

type timeRange struct {
	MinTimeMS, MaxTimeMS uint64
	ExtHash              extHash
}

func (timeRange) Type() string { return "timerange" } // xxx "timerange1"?

func newTimeRange(minTimeMS, maxTimeMS uint64) *entry {
	return &entry{
		body: &timeRange{
			MinTimeMS: minTimeMS,
			MaxTimeMS: maxTimeMS,
		},
	}
}
