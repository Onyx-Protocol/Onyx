package tx

import "chain/protocol/bc"

type output struct {
	body struct {
		Source         valueSource
		ControlProgram program
		Data           bc.Hash
		ExtHash        bc.Hash
	}
	ordinal int
}

func (output) Type() string         { return "output1" }
func (o *output) Body() interface{} { return o.body }

func (o output) Ordinal() int { return o.ordinal }

func newOutput(source valueSource, controlProgram program, data bc.Hash, ordinal int) *output {
	out := new(output)
	out.body.Source = source
	out.body.ControlProgram = controlProgram
	out.body.Data = data
	out.ordinal = ordinal
	return out
}
