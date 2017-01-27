package tx

type header struct {
	Version              uint64
	Results              []entryRef
	Data                 entryRef
	MinTimeMS, MaxTimeMS uint64
	ExtHash              extHash
}

func (header) Type() string { return "txheader" }

func newHeader(version uint64, results []entryRef, data entryRef, minTimeMS, maxTimeMS uint64) *entry {
	return &entry{
		body: &header{
			Version:   version,
			Results:   results,
			Data:      data,
			MinTimeMS: minTimeMS,
			MaxTimeMS: maxTimeMS,
		},
	}
}
