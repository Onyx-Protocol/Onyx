package bc

type retirement struct {
	body struct {
		Source  valueSource
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to body.Source.
	Source entry // *issuance, *spend, or *mux
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func (r retirement) Ordinal() int { return r.ordinal }

func newRetirement(data Hash, ordinal int) *retirement {
	r := new(retirement)
	r.body.Data = data
	r.ordinal = ordinal
	return r
}

func (r *retirement) setSource(e entry, value AssetAmount, position uint64) {
	r.setSourceID(entryID(e), value, position)
	r.Source = e
}

func (r *retirement) setSourceID(sourceID Hash, value AssetAmount, position uint64) {
	r.body.Source = valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	r.Source = nil
}
