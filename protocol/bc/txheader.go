package bc

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.

func (TxHeader) Type() string         { return "txheader" }
func (h *TxHeader) body() interface{} { return h.Body }

func (TxHeader) Ordinal() int { return -1 }

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version uint64, results []Entry, data Hash, minTimeMS, maxTimeMS uint64) *TxHeader {
	h := new(TxHeader)
	h.Body.Version = version
	h.Body.Data = data
	h.Body.MinTimeMS = minTimeMS
	h.Body.MaxTimeMS = maxTimeMS

	h.Results = results
	for _, r := range results {
		h.Body.ResultIDs = append(h.Body.ResultIDs, EntryID(r))
	}

	return h
}
