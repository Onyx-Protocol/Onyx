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
}

func (header) Type() string         { return "txheader" }
func (h *header) Body() interface{} { return h.body }

func (header) Ordinal() int { return -1 }

func newHeader(version uint64, results []bc.Hash, data bc.Hash, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.Version = version
	h.body.Results = results
	h.body.Data = data
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS
	return h
}
