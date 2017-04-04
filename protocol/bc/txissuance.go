package bc

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.
//
// (Not to be confused with the deprecated type IssuanceInput.)

func (Issuance) Type() string           { return "issuance1" }
func (iss *Issuance) body() interface{} { return iss.Body }

func (iss *Issuance) SetDestination(id Hash, val AssetAmount, pos uint64, e Entry) {
	iss.Witness.Destination = &ValueDestination{
		Ref:      id.Proto(),
		Value:    val.Proto(),
		Position: pos,
	}
}

// NewIssuance creates a new Issuance.
func NewIssuance(anchor Entry, value AssetAmount, data Hash, ordinal uint64) *Issuance {
	return &Issuance{
		Body: &Issuance_Body{
			AnchorId: EntryID(anchor).Proto(),
			Value:    value.Proto(),
			Data:     data.Proto(),
		},
		Witness: &Issuance_Witness{},
		Ordinal: ordinal,
	}
}
