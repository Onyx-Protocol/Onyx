package tx

type data struct {
	body    []byte
	extHash extHash
}

func (data) Type() string         { return "data1" }
func (d *data) Body() interface{} { return d.body }

func newData(refData []byte) entry {
	d := new(data)
	d.body = refData
	return d
}
