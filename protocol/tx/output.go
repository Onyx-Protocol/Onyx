package tx

type output struct {
	content struct {
		source         valueSource
		controlProgram program
		reference      entryRef
		extHash        extHash
	}
}

func (output) Type() string            { return "output1" }
func (o *output) Content() interface{} { return &o.content }
