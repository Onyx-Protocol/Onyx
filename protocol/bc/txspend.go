package bc

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)

func (Spend) Type() string         { return "spend1" }
func (s *Spend) body() interface{} { return s.Body }

func (s Spend) Ordinal() int { return s.ordinal }

func (s *Spend) SetDestination(id Hash, val AssetAmount, pos uint64, e Entry) {
	s.Witness.Destination = ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
		Entry:    e,
	}
}

// NewSpend creates a new Spend.
func NewSpend(out *Output, data Hash, ordinal int) *Spend {
	s := new(Spend)
	s.Body.SpentOutputID = EntryID(out)
	s.Body.Data = data
	s.ordinal = ordinal
	s.SpentOutput = out
	return s
}

func (s *Spend) SetAnchored(id Hash, entry Entry) {
	s.Witness.AnchoredID = id
	s.Anchored = entry
}
