package bc

type Spend struct {
	body struct {
		SpentOutput Hash // the hash of an output entry
		Data        Hash
		ExtHash     Hash
	}
	ordinal int

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to body.SpentOutput.
	SpentOutput *Output
}

func (Spend) Type() string         { return "spend1" }
func (s *Spend) Body() interface{} { return s.body }

func (s Spend) Ordinal() int { return s.ordinal }

func newSpend(out *Output, data Hash, ordinal int) *Spend {
	s := new(Spend)
	s.body.SpentOutput = EntryID(out)
	s.body.Data = data
	s.ordinal = ordinal
	s.SpentOutput = out
	return s
}
