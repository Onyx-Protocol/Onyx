package bc

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)
type Spend struct {
	Body struct {
		SpentOutputID Hash // the hash of an output entry
		Data          Hash
		ExtHash       Hash
	}
	ordinal int

	Witness struct {
		Destination ValueDestination
		Arguments   [][]byte
		AnchoredID  Hash
	}

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to Body.SpentOutputID.
	SpentOutput *Output

	// Anchored contains a pointer to the manifested entry corresponding
	// to witness.AnchoredID.
	Anchored Entry
}

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
