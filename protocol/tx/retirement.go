package tx

import "chain/protocol/bc"

type retirement struct {
	body struct {
		Source  valueSource
		Data    bc.Hash
		ExtHash bc.Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to body.Source.
	Source entry // *issuance, *spend, or *mux
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func (r retirement) Ordinal() int { return r.ordinal }

func newRetirement(data bc.Hash, ordinal int) *retirement {
	r := new(retirement)
	r.body.Data = data
	r.ordinal = ordinal
	return r
}

func (r *retirement) setSource(e entry, value bc.AssetAmount, position uint64) {
	r.setSourceID(entryID(e), value, position)
	r.Source = e
}

func (r *retirement) setSourceID(sourceID bc.Hash, value bc.AssetAmount, position uint64) {
	r.body.Source = valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	r.Source = nil
}
