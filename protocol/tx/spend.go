package tx

import "chain/protocol/bc"

type spend struct {
	body struct {
		SpentOutput bc.Hash // the hash of an output entry
		Data        bc.Hash
		ExtHash     bc.Hash
	}
	ordinal int

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to body.SpentOutput.
	SpentOutput *output
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func (s spend) Ordinal() int { return s.ordinal }

func newSpend(out *output, data bc.Hash, ordinal int) *spend {
	s := new(spend)
	s.body.SpentOutput = entryID(out)
	s.body.Data = data
	s.ordinal = ordinal
	s.SpentOutput = out
	return s
}
