package tx

type retirement struct {
	body struct {
		source  valueSource
		data    entryRef
		extHash extHash
	}
}

func (retirement) Type() string         { return "retirement1" }
func (r *retirement) Body() interface{} { return r.body }

func newRetirement(source valueSource, data entryRef) *retirement {
	r := new(retirement)
	r.body.source = source
	r.body.data = data
	return r
}
