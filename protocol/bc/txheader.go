package bc

type TxHeader struct {
	body struct {
		Version              uint64
		Results              []Hash
		Data                 Hash
		MinTimeMS, MaxTimeMS uint64
		ExtHash              Hash
	}

	// Results contains (pointers to) the manifested entries for the
	// items in body.Results.
	Results []Entry // each entry is *output or *retirement
}

func (TxHeader) Type() string         { return "txheader" }
func (h *TxHeader) Body() interface{} { return h.body }

func (TxHeader) Ordinal() int { return -1 }

func newHeader(version uint64, results []Entry, data Hash, minTimeMS, maxTimeMS uint64) *TxHeader {
	h := new(TxHeader)
	h.body.Version = version
	h.body.Data = data
	h.body.MinTimeMS = minTimeMS
	h.body.MaxTimeMS = maxTimeMS

	h.Results = results
	for _, r := range results {
		h.body.Results = append(h.body.Results, EntryID(r))
	}

	return h
}
