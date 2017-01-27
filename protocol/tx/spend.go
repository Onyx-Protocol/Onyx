package tx

type spend struct {
	body struct {
		SpentOutput entryRef // must be an Output entry
		Data        entryRef // must be a Data entry
		ExtHash     extHash
	}
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func newSpend(spentOutput, data entryRef) entry {
	s := new(spend)
	s.body.SpentOutput = spentOutput
	s.body.Data = data
	return s
}
