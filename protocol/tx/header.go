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

func newHeader(version uint64, results []entry, data bc.Hash, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.Version = version
	h.body.Data = data
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS

	h.Results = results
	for _, r := range results {
		h.body.Results = append(h.body.Results, entryID(r))
	}

	return h
}
