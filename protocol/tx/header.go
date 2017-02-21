package tx

import "chain/protocol/bc"

type header struct {
	body struct {
		Version              uint64
		Results              []entryRef
		RefDataHash          bc.Hash
		MinTimeMS, MaxTimeMS uint64
		ExtHash              extHash
	}
}

func (header) Type() string         { return "txheader" }
func (h *header) Body() interface{} { return h.body }

func (header) Ordinal() int { return -1 }

func newHeader(version uint64, results []entryRef, refDataHash bc.Hash, minTimeMS, maxTimeMS uint64) *header {
	h := new(header)
	h.body.Version = version
	h.body.Results = results
	h.body.RefDataHash = refDataHash
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS
	return h
}
