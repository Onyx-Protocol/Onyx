package bc

// Retirement is for the permanent removal of some value from a
// blockchain. The value it contains can never be obtained by later
// entries. Retirement satisfies the Entry interface.

func (Retirement) Type() string         { return "retirement1" }
func (r *Retirement) body() interface{} { return r.Body }

// NewRetirement creates a new Retirement.
func NewRetirement(source *ValueSource, data Hash, ordinal uint64) *Retirement {
	r := new(Retirement)
	r.Body.Source = source
	r.Body.Data = data.Proto()
	r.Ordinal = ordinal
	return r
}
