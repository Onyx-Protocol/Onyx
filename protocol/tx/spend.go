package tx

import "chain/protocol/bc"

type spend struct {
	body struct {
		SpentOutput entryRef // must be an Output entry
		RefDataHash bc.Hash
		ExtHash     extHash
	}
	ordinal int
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func (s spend) Ordinal() int { return s.ordinal }

func newSpend(spentOutput entryRef, refDataHash bc.Hash, ordinal int) *spend {
	s := new(spend)
	s.body.SpentOutput = spentOutput
	s.body.RefDataHash = refDataHash
	s.ordinal = ordinal
	return s
}
