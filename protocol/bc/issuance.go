package bc

import "io"

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.
//
// (Not to be confused with the deprecated type IssuanceInput.)

func (Issuance) typ() string { return "issuance1" }
func (iss *Issuance) writeForHash(w io.Writer) {
	mustWriteForHash(w, iss.AnchorId)
	mustWriteForHash(w, iss.Value)
	mustWriteForHash(w, iss.Data)
	mustWriteForHash(w, iss.ExtHash)
}

func (iss *Issuance) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	iss.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewIssuance creates a new Issuance.
func NewIssuance(anchorID *Hash, value *AssetAmount, data *Hash, ordinal uint64) *Issuance {
	return &Issuance{
		AnchorId: anchorID,
		Value:    value,
		Data:     data,
		Ordinal:  ordinal,
	}
}
