package bc

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)

func (Spend) Type() string         { return "spend1" }
func (s *Spend) body() interface{} { return s.Body }

func (s *Spend) SetDestination(id Hash, val AssetAmount, pos uint64) {
	s.Witness.Destination = &ValueDestination{
		Ref:      id.Proto(),
		Value:    val.Proto(),
		Position: pos,
	}
}

// NewSpend creates a new Spend.
func NewSpend(out *Output, data Hash, ordinal uint64) *Spend {
	return &Spend{
		Body: &Spend_Body{
			SpentOutputId: EntryID(out).Proto(),
			Data:          data.Proto(),
		},
		Witness: &Spend_Witness{},
		Ordinal: ordinal,
	}
}

func (s *Spend) SetAnchored(id Hash) {
	s.Witness.AnchoredId = id.Proto()
}
