package bc

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.
//
// (Not to be confused with the deprecated type IssuanceInput.)

func (Issuance) Type() string           { return "issuance1" }
func (iss *Issuance) body() interface{} { return iss.Body }

func (iss *Issuance) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	iss.Witness.Destination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewIssuance creates a new Issuance.
func NewIssuance(anchorID *Hash, value *AssetAmount, data *Hash, ordinal uint64) *Issuance {
	return &Issuance{
		Body: &Issuance_Body{
			AnchorId: anchorID,
			Value:    value,
			Data:     data,
		},
		Witness: &Issuance_Witness{},
		Ordinal: ordinal,
	}
}
