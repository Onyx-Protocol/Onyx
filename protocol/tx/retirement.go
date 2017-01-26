package tx

type retirement struct {
	body struct {
		Source  valueSource
		Data    entryRef
		ExtHash extHash
	}
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func newRetirement(source valueSource, data entryRef) *retirement {
	r := new(retirement)
	r.body.Source = source
	r.body.Data = data
	return r
}
