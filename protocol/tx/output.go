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
func (o *output) Body() interface{} { return o.body }

func newOutput(source valueSource, controlProgram program, reference entryRef) *output {
	out := new(output)
	out.body.source = source
	out.body.controlProgram = controlProgram
	out.body.reference = reference
	return out
}
