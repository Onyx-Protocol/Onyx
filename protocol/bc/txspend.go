package bc

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)
type Spend struct {
	Body struct {
		SpentOutput Hash // the hash of an output entry
		Data        Hash
		ExtHash     Hash
	}
	ordinal int

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to Body.SpentOutput.
	SpentOutput *Output
}

func (Spend) Type() string         { return "spend1" }
func (s *Spend) body() interface{} { return s.Body }

func (s Spend) Ordinal() int { return s.ordinal }

// NewSpend creates a new Spend.
func NewSpend(out *Output, data Hash, ordinal int) *Spend {
	s := new(Spend)
	s.Body.SpentOutput = EntryID(out)
	s.Body.Data = data
	s.ordinal = ordinal
	s.SpentOutput = out
	return s
}
