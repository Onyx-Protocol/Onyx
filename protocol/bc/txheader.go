package bc

import "io"

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.

func (TxHeader) typ() string { return "txheader" }
func (h *TxHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, h.Version)
	mustWriteForHash(w, h.ResultIds)
	mustWriteForHash(w, h.Data)
	mustWriteForHash(w, h.MinTimeMs)
	mustWriteForHash(w, h.MaxTimeMs)
	mustWriteForHash(w, h.ExtHash)
}

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version uint64, resultIDs []*Hash, data *Hash, minTimeMS, maxTimeMS uint64) *TxHeader {
	return &TxHeader{
		Version:   version,
		ResultIds: resultIDs,
		Data:      data,
		MinTimeMs: minTimeMS,
		MaxTimeMs: maxTimeMS,
	}
}
