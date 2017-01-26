package tx

type header struct {
	body struct {
		version              uint64
		results              []entryRef
		data                 entryRef
		references           []entryRef
		minTimeMS, maxTimeMS uint64
		extHash              extHash
	}
}

func (header) Type() string         { return "header" }
func (h *header) Body() interface{} { return h.body }

func newHeader(version uint64, results []entryRef, data entryRef, references []entryRef, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.version = version
	h.body.results = results
	h.body.data = data
	h.body.references = references
	h.body.minTimeMS = minTimeMS
	h.body.maxTimeMS = maxTimeMS
	return h
}
