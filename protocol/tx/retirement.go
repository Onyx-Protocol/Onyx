package tx

import "chain/protocol/bc"

type retirement struct {
	body struct {
		Source      valueSource
		RefDataHash bc.Hash
		ExtHash     extHash
	}
	ordinal int
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func (r retirement) Ordinal() int { return r.ordinal }

func newRetirement(source valueSource, refDataHash bc.Hash, ordinal int) *retirement {
	r := new(retirement)
	r.body.Source = source
	r.body.RefDataHash = refDataHash
	r.ordinal = ordinal
	return r
}
