package tx

type retirement struct {
	body struct {
		Source  valueSource
		Data    entryRef
		ExtHash extHash
	}
	ordinal int
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func (r retirement) Ordinal() int { return r.ordinal }

func newRetirement(source valueSource, data entryRef, ordinal int) *retirement {
	r := new(retirement)
	r.body.Source = source
	r.body.Data = data
	r.ordinal = ordinal
	return r
}
