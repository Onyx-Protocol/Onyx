package bc

// Retirement is for the permanent removal of some value from a
// blockchain. The value it contains can never be obtained by later
// entries. Retirement satisfies the Entry interface.
type Retirement struct {
	Body struct {
		Source  ValueSource
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to Body.Source.
	Source Entry // *issuance, *spend, or *mux
}

func (Retirement) Type() string         { return "retirement1" }
func (r *Retirement) body() interface{} { return r.Body }

func (r Retirement) Ordinal() int { return r.ordinal }

// NewRetirement creates a new Retirement. Once created, its source
// should be set with setSource or setSourceID.
func NewRetirement(data Hash, ordinal int) *Retirement {
	r := new(Retirement)
	r.Body.Data = data
	r.ordinal = ordinal
	return r
}

func (r *Retirement) setSource(e Entry, value AssetAmount, position uint64) {
	r.setSourceID(EntryID(e), value, position)
	r.Source = e
}

func (r *Retirement) setSourceID(sourceID Hash, value AssetAmount, position uint64) {
	r.Body.Source = ValueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	r.Source = nil
}
