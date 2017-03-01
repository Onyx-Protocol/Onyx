package tx

import "chain/protocol/bc"

type header struct {
	body struct {
		Version              uint64
		Results              []bc.Hash
		Data                 bc.Hash
		MinTimeMS, MaxTimeMS uint64
		ExtHash              bc.Hash
	}

	// Results contains (pointers to) the manifested entries for the
	// items in body.Results.
	Results []entry // each entry is *output or *retirement
}

func (header) Type() string         { return "txheader" }
func (h *header) Body() interface{} { return h.body }

func (header) Ordinal() int { return -1 }

func newHeader(version uint64, data bc.Hash, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.Version = version
	h.body.Data = data
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS

	return h
}

func (h *header) addResult(e entry) {
	w := newIDWrapper(e, nil)
	h.body.Results = append(h.body.Results, w.Hash)
	h.Results = append(h.Results, w)
}
