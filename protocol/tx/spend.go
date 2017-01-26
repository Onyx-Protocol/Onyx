package tx

type spend struct {
	body struct {
		SpentOutput entryRef // must be an Output entry
		reference   entryRef // must be a Data entry
		extHash     extHash
	}
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return &s.body }

func newSpend(spentOutput, reference, destination entryRef) entry {
	s := new(spend)
	s.body.SpentOutput = spentOutput
	s.body.reference = reference
	return s
}
