package tx

type spend struct {
	body struct {
		SpentOutput entryRef // must be an Output entry
		Data        entryRef // must be a Data entry
		ExtHash     extHash
	}
	ordinal int
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func (s spend) Ordinal() int { return s.ordinal }

func newSpend(spentOutput, data entryRef, ordinal int) *spend {
	s := new(spend)
	s.body.SpentOutput = spentOutput
	s.body.Data = data
	s.ordinal = ordinal
	return s
}
