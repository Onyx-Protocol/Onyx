package bc

import "chain/errors"

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.
type TxHeader struct {
	Body struct {
		Version              uint64
		ResultIDs            []Hash
		Data                 Hash
		MinTimeMS, MaxTimeMS uint64
		ExtHash              Hash
	}

	// Results contains (pointers to) the manifested entries for the
	// items in Body.ResultIDs.
	Results []Entry // each entry is *output or *retirement
}

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

// checkValid does only part of the work of validating a tx header. The block-related parts of tx validation are in ValidateBlock.
func (tx *TxHeader) checkValid(vs *validationState) error {
	if tx.Body.MaxTimeMS > 0 {
		if tx.Body.MaxTimeMS < tx.Body.MinTimeMS {
			return errors.WithDetailf(errBadTimeRange, "min time %d, max time %d", tx.Body.MinTimeMS, tx.Body.MaxTimeMS)
		}
	}

	for i, resID := range tx.Body.ResultIDs {
		res := tx.Results[i]
		vs2 := *vs
		vs2.entryID = resID
		err := res.checkValid(&vs2)
		if err != nil {
			return errors.Wrapf(err, "checking result %d", i)
		}
	}

	if tx.Body.Version == 1 {
		if len(tx.Body.ResultIDs) == 0 {
			return errEmptyResults
		}

		if tx.Body.ExtHash != (Hash{}) {
			return errNonemptyExtHash
		}
	}

	return nil
}
