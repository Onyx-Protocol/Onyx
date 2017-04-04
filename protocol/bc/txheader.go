package bc

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.

func (TxHeader) Type() string         { return "txheader" }
func (h *TxHeader) body() interface{} { return h.Body }

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version uint64, resultIDs []Hash, data Hash, minTimeMS, maxTimeMS uint64) *TxHeader {
	result := &TxHeader{
		Body: &TxHeader_Body{
			Version: version,
			Data: data.Proto(),
			MinTimeMs: minTimeMS,
			MaxTimeMs: maxTimeMS,
		},
	}
	for _, id := range resultIDs {
		result.Body.ResultIds = append(result.Body.ResultIds, id.Proto())
	}
	return result
}
