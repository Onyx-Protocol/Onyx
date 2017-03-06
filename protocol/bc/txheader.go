package bc

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.
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

func (h *TxHeader) Version() uint64 {
	return h.body.Version
}

func (h *TxHeader) Data() Hash {
	return h.body.Data
}

func (h *TxHeader) ResultID(n uint32) Hash {
	return h.body.Results[n]
}

func (h *TxHeader) MinTimeMS() uint64 {
	return h.body.MinTimeMS
}

func (h *TxHeader) MaxTimeMS() uint64 {
	return h.body.MaxTimeMS
}

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version uint64, results []Entry, data Hash, minTimeMS, maxTimeMS uint64) *TxHeader {
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
