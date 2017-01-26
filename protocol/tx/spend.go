package tx

type spend struct {
	body struct {
		spentOutput entryRef // must be an Output entry
		reference   entryRef // must be a Data entry
		extHash     extHash
	}
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func newSpend(spentOutput, reference entryRef) entry {
	s := new(spend)
	s.body.spentOutput = spentOutput
	s.body.reference = reference
	return s
}
