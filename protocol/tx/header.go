package tx

type header struct {
	body struct {
		Version              uint64
		Results              []entryRef
		Data                 entryRef
		References           []entryRef
		MinTimeMS, MaxTimeMS uint64
		ExtHash              extHash
	}
}

func (header) Type() string         { return "header" }
func (h *header) Body() interface{} { return h.body }

func newHeader(version uint64, results []entryRef, data entryRef, references []entryRef, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.Version = version
	h.body.Results = results
	h.body.Data = data
	h.body.References = references
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS
	return h
}
