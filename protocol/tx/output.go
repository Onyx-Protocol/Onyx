package tx

type output struct {
	body struct {
		source         valueSource
		controlProgram program
		reference      entryRef
		extHash        extHash
	}
}

func (output) Type() string         { return "output1" }
func (o *output) Body() interface{} { return &o.body }
