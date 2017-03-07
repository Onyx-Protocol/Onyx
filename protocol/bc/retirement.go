package bc

type Retirement struct {
	body struct {
		Source  valueSource
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to body.Source.
	Source Entry // *issuance, *spend, or *mux
}

func (Retirement) Type() string         { return "retirement1" }
func (r *Retirement) Body() interface{} { return r.body }

func (r Retirement) Ordinal() int { return r.ordinal }

func newRetirement(data Hash, ordinal int) *Retirement {
	r := new(Retirement)
	r.body.Data = data
	r.ordinal = ordinal
	return r
}

func (r *Retirement) setSource(e Entry, value AssetAmount, position uint64) {
	r.setSourceID(EntryID(e), value, position)
	r.Source = e
}

func (r *Retirement) setSourceID(sourceID Hash, value AssetAmount, position uint64) {
	r.body.Source = valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	r.Source = nil
}
