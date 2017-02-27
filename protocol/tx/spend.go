package tx

import "chain/protocol/bc"

type spend struct {
	body struct {
		SpentOutput bc.Hash // must be an Output entry
		Data        bc.Hash
		ExtHash     bc.Hash
	}
	ordinal int
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func (s spend) Ordinal() int { return s.ordinal }

func newSpend(spentOutput bc.Hash, data bc.Hash, ordinal int) *spend {
	s := new(spend)
	s.body.SpentOutput = spentOutput
	s.body.Data = data
	s.ordinal = ordinal
	return s
}
