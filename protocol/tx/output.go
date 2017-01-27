package tx

type output struct {
	body struct {
		Source         valueSource
		ControlProgram program
		Data           entryRef
		ExtHash        extHash
	}
}

func (output) Type() string         { return "output1" }
func (o *output) Body() interface{} { return o.body }

func newOutput(source valueSource, controlProgram program, data entryRef) *output {
	out := new(output)
	out.body.Source = source
	out.body.ControlProgram = controlProgram
	out.body.Data = data
	return out
}
