package tx

type spend struct {
	content struct {
		SpentOutput entryRef // must be an Output entry
		reference   entryRef // must be a Data entry
		extHash     extHash
	}
}

func (spend) Type() string            { return "spend1" }
func (s *spend) Content() interface{} { return &s.content }

func newSpend(spentOutput, reference, destination entryRef) entry {
	s := new(spend)
	s.content.SpentOutput = spentOutput
	s.content.reference = reference
	return s
}
