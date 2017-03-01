package tx

import "chain/protocol/bc"

type spend struct {
	body struct {
		SpentOutput bc.Hash // must be an Output entry
		Data        bc.Hash
		ExtHash     bc.Hash
	}
	ordinal int

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to body.SpentOutput. It may be nil, in which case
	// spentInfo must be non-nil.
	SpentOutput *output

	// spentInfo contains validation-essential data in case the spent
	// output object is not present (in SpentOutput).
	spentInfo *spentInfo
}

type spentInfo struct {
	bc.AssetAmount
	program
}

func (spend) Type() string         { return "spend1" }
func (s *spend) Body() interface{} { return s.body }

func (s spend) Ordinal() int { return s.ordinal }

func newSpend(data bc.Hash, ordinal int) *spend {
	s := new(spend)
	s.body.Data = data
	s.ordinal = ordinal
	return s
}

func (s *spend) setSpentOutput(o *output) {
	s.body.SpentOutput = entryID(o)
	s.SpentOutput = o
}

func (s *spend) setSpentInfo(spentOutputID bc.Hash, value bc.AssetAmount, program program) {
	s.body.SpentOutput = spentOutputID
	s.spentInfo = &spentInfo{
		AssetAmount: value,
		program:     program,
	}
}
